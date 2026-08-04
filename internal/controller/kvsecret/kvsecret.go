package kvsecret

import (
	"context"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/kvsecret/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotKVSecret    = "managed resource is not a KVSecret custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
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
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()),
		managed.WithDeterministicExternalName(true))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.KVSecret{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.KVSecret)
	if !ok {
		return nil, errors.New(errNotKVSecret)
	}

	if err := clients.TrackUsage(ctx, c.kube, cr); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	svc, err := clients.Connect(ctx, c.kube, cr)
	if err != nil {
		return nil, err
	}

	return &external{service: svc}, nil
}

type external struct {
	service *clients.VaultClient
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.KVSecret)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotKVSecret)
	}

	data, err := e.service.GetKVSecret(ctx, cr.Spec.ForProvider.Path, cr.Spec.ForProvider.MountPath, cr.Spec.ForProvider.Version)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
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
	if err := e.service.CreateKVSecret(ctx, cr.Spec.ForProvider.Path, cr.Spec.ForProvider.MountPath, cr.Spec.ForProvider.Data, cr.Spec.ForProvider.Version); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateKVSecret)
	}
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.KVSecret)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotKVSecret)
	}
	if err := e.service.UpdateKVSecret(ctx, cr.Spec.ForProvider.Path, cr.Spec.ForProvider.MountPath, cr.Spec.ForProvider.Data, cr.Spec.ForProvider.Version); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateKVSecret)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.KVSecret)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotKVSecret)
	}
	if err := e.service.DeleteKVSecret(ctx, cr.Spec.ForProvider.Path, cr.Spec.ForProvider.MountPath, cr.Spec.ForProvider.Version); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteKVSecret)
	}
	return managed.ExternalDelete{}, nil
}
