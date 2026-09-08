package authmethod

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/rossigee/provider-vault/internal/features"

	v1beta1 "github.com/rossigee/provider-vault/apis/authmethod/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotAuthMethod    = "managed resource is not an AuthMethod custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errCreateAuthMethod = "cannot create Vault auth method"
	errDeleteAuthMethod = "cannot delete Vault auth method"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.AuthMethodKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube: mgr.GetClient(),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()),
		managed.WithDeterministicExternalName(true),
	}
	if o.Features.Enabled(features.EnableAlphaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.AuthMethodGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.AuthMethod{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.AuthMethod)
	if !ok {
		return nil, errors.New(errNotAuthMethod)
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
	cr, ok := mg.(*v1beta1.AuthMethod)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAuthMethod)
	}

	data, err := e.service.GetAuthMethod(ctx, cr.Spec.ForProvider.MountPath)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.MountPath = cr.Spec.ForProvider.MountPath

	upToDate := !clients.DriftedString(data, "type", cr.Spec.ForProvider.Type)

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
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

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.AuthMethod)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAuthMethod)
	}
	if err := e.service.DisableAuthMethod(ctx, cr.Spec.ForProvider.MountPath); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAuthMethod)
	}
	return managed.ExternalDelete{}, nil
}
