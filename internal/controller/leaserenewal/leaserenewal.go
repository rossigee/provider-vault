package leaserenewal

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/leaserenewal/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotLeaseRenewal  = "managed resource is not a LeaseRenewal custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errLookupLease      = "cannot lookup Vault lease"
	errRenewLease       = "cannot renew Vault lease"
	errRevokeLease      = "cannot revoke Vault lease"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.LeaseRenewalKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.LeaseRenewalGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube: mgr.GetClient(),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()),
		managed.WithDeterministicExternalName(true))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.LeaseRenewal{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.LeaseRenewal)
	if !ok {
		return nil, errors.New(errNotLeaseRenewal)
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
	cr, ok := mg.(*v1beta1.LeaseRenewal)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLeaseRenewal)
	}

	p := cr.Spec.ForProvider

	info, err := e.service.LookupLease(ctx, p.LeaseID)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.LeaseID = info.LeaseID
	cr.Status.AtProvider.Renewable = info.Renewable
	cr.Status.AtProvider.TTL = info.LeaseDuration

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.LeaseRenewal)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLeaseRenewal)
	}

	p := cr.Spec.ForProvider

	info, err := e.service.RenewLease(ctx, p.LeaseID, p.Increment)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errRenewLease)
	}

	cr.Status.AtProvider.LeaseID = info.LeaseID
	cr.Status.AtProvider.Renewable = info.Renewable
	cr.Status.AtProvider.TTL = info.LeaseDuration
	now := metav1.Now()
	cr.Status.AtProvider.LastRenewed = &now

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.LeaseRenewal)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLeaseRenewal)
	}

	p := cr.Spec.ForProvider

	info, err := e.service.RenewLease(ctx, p.LeaseID, p.Increment)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errRenewLease)
	}

	cr.Status.AtProvider.LeaseID = info.LeaseID
	cr.Status.AtProvider.Renewable = info.Renewable
	cr.Status.AtProvider.TTL = info.LeaseDuration
	now := metav1.Now()
	cr.Status.AtProvider.LastRenewed = &now

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.LeaseRenewal)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotLeaseRenewal)
	}

	p := cr.Spec.ForProvider

	revokeOnDelete := false
	if p.RevokeOnDelete != nil {
		revokeOnDelete = *p.RevokeOnDelete
	}

	if revokeOnDelete {
		if err := e.service.RevokeLease(ctx, p.LeaseID); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errRevokeLease)
		}
	}

	return managed.ExternalDelete{}, nil
}
