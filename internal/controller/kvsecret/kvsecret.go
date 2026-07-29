package kvsecret

import (
	"context"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/kvsecret/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotKVSecret    = "managed resource is not a KVSecret custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetCreds       = "cannot get credentials"
	errCreateKVSecret = "cannot create KV secret"
	errDeleteKVSecret = "cannot delete KV secret"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.KVSecretKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.KVSecretGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error { return nil }),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.KVSecret{}).
		Complete(r)
}

type connector struct {
	kube  client.Client
	usage resource.TrackerFn
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.KVSecret)
	if !ok {
		return nil, errors.New(errNotKVSecret)
	}

	pc := &vaultv1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: cr.GetProviderConfigRef().Name, Namespace: cr.GetNamespace()}, pc); err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	vaultClient, err := clients.NewClientFromProviderConfig(ctx, c.kube, pc)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	return &external{service: vaultClient, kube: c.kube}, nil
}

type external struct {
	service *clients.VaultClient
	kube    client.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.KVSecret)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotKVSecret)
	}

	data, err := e.service.GetKVSecret(ctx, cr.Spec.ForProvider.Path, cr.Spec.ForProvider.MountPath)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.Data = data
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: cmp.Equal(cr.Spec.ForProvider.Data, data),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.KVSecret)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotKVSecret)
	}
	if err := e.service.CreateKVSecret(ctx, cr.Spec.ForProvider.Path, cr.Spec.ForProvider.MountPath, cr.Spec.ForProvider.Data); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateKVSecret)
	}
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.KVSecret)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotKVSecret)
	}
	if err := e.service.CreateKVSecret(ctx, cr.Spec.ForProvider.Path, cr.Spec.ForProvider.MountPath, cr.Spec.ForProvider.Data); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateKVSecret)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1beta1.KVSecret)
	if !ok {
		return errors.New(errNotKVSecret)
	}
	return errors.Wrap(e.service.DeleteKVSecret(ctx, cr.Spec.ForProvider.Path, cr.Spec.ForProvider.MountPath), errDeleteKVSecret)
}
