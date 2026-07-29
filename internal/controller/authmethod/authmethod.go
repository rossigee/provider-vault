package authmethod

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

	v1beta1 "github.com/rossigee/provider-vault/apis/authmethod/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotAuthMethod    = "managed resource is not an AuthMethod custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetCreds         = "cannot get credentials"
	errCreateAuthMethod = "cannot create Vault auth method"
	errDeleteAuthMethod = "cannot delete Vault auth method"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.AuthMethodKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.AuthMethodGroupVersionKind),
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
		For(&v1beta1.AuthMethod{}).
		Complete(r)
}

type connector struct {
	kube  client.Client
	usage resource.TrackerFn
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.AuthMethod)
	if !ok {
		return nil, errors.New(errNotAuthMethod)
	}

	pc := &vaultv1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: cr.GetProviderConfigRef().Name, Namespace: cr.GetNamespace()}, pc); err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	vaultClient, err := clients.NewClientFromProviderConfig(ctx, c.kube, pc)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	return &external{service: vaultClient}, nil
}

type external struct {
	service *clients.VaultClient
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.AuthMethod)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAuthMethod)
	}

	_, err := e.service.GetAuthMethod(ctx, cr.Spec.ForProvider.MountPath)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.MountPath = cr.Spec.ForProvider.MountPath
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.AuthMethod)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAuthMethod)
	}
	if err := e.service.EnableAuthMethod(ctx, cr.Spec.ForProvider.MountPath, cr.Spec.ForProvider.Type, cr.Spec.ForProvider.Config); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAuthMethod)
	}
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.AuthMethod)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAuthMethod)
	}
	if err := e.service.EnableAuthMethod(ctx, cr.Spec.ForProvider.MountPath, cr.Spec.ForProvider.Type, cr.Spec.ForProvider.Config); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateAuthMethod)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1beta1.AuthMethod)
	if !ok {
		return errors.New(errNotAuthMethod)
	}
	return errors.Wrap(e.service.DisableAuthMethod(ctx, cr.Spec.ForProvider.MountPath), errDeleteAuthMethod)
}
