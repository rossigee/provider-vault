package namespaces

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/rossigee/provider-vault/internal/features"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v1beta1 "github.com/rossigee/provider-vault/apis/namespaces/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotVaultNamespace = "managed resource is not a VaultNamespace custom resource"
	errTrackPCUsage      = "cannot track ProviderConfig usage"
	errGetPC             = "cannot get ProviderConfig"
	errGetCreds          = "cannot get credentials"
	errCreateNamespace   = "cannot create Vault namespace"
	errDeleteNamespace   = "cannot delete Vault namespace"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.VaultNamespaceKind)

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
		resource.ManagedKind(v1beta1.VaultNamespaceGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.VaultNamespace{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.VaultNamespace)
	if !ok {
		return nil, errors.New(errNotVaultNamespace)
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
	cr, ok := mg.(*v1beta1.VaultNamespace)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotVaultNamespace)
	}

	p := cr.Spec.ForProvider

	_, err := e.service.GetNamespace(ctx, p.Name)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Name = p.Name

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.VaultNamespace)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotVaultNamespace)
	}

	p := cr.Spec.ForProvider

	if err := e.service.CreateNamespace(ctx, p.Name, p.Description); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateNamespace)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.VaultNamespace)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotVaultNamespace)
	}

	p := cr.Spec.ForProvider

	if err := e.service.CreateNamespace(ctx, p.Name, p.Description); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateNamespace)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.VaultNamespace)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotVaultNamespace)
	}

	p := cr.Spec.ForProvider

	if err := e.service.DeleteNamespace(ctx, p.Name); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteNamespace)
	}

	return managed.ExternalDelete{}, nil
}
