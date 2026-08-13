package quota

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v1beta1 "github.com/rossigee/provider-vault/apis/quota/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotQuota     = "managed resource is not a Quota custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errCreateQuota  = "cannot create Vault quota"
	errDeleteQuota  = "cannot delete Vault quota"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.QuotaKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.QuotaGroupVersionKind),
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
		For(&v1beta1.Quota{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.Quota)
	if !ok {
		return nil, errors.New(errNotQuota)
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
	cr, ok := mg.(*v1beta1.Quota)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotQuota)
	}

	p := cr.Spec.ForProvider

	data, err := e.service.GetQuota(ctx, p.Type, p.Name)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Name = p.Name
	cr.Status.AtProvider.Type = p.Type

	_ = data

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Quota)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotQuota)
	}

	p := cr.Spec.ForProvider

	switch p.Type {
	case "rate":
		rate := "0"
		if p.Rate != "" {
			rate = p.Rate
		}
		interval := p.Interval
		if interval == "" {
			interval = "second"
		}
		if err := e.service.CreateRateQuota(ctx, p.Name, p.Path, rate, interval, p.Blocked); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateQuota)
		}
	case "lease":
		maxLeases := 0
		if p.MaxLeases != nil {
			maxLeases = *p.MaxLeases
		}
		if err := e.service.CreateLeaseQuota(ctx, p.Name, p.Path, maxLeases, p.Blocked); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateQuota)
		}
	default:
		return managed.ExternalCreation{}, errors.Errorf("unknown quota type: %s", p.Type)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Quota)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotQuota)
	}

	p := cr.Spec.ForProvider

	switch p.Type {
	case "rate":
		rate := "0"
		if p.Rate != "" {
			rate = p.Rate
		}
		interval := p.Interval
		if interval == "" {
			interval = "second"
		}
		if err := e.service.CreateRateQuota(ctx, p.Name, p.Path, rate, interval, p.Blocked); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errCreateQuota)
		}
	case "lease":
		maxLeases := 0
		if p.MaxLeases != nil {
			maxLeases = *p.MaxLeases
		}
		if err := e.service.CreateLeaseQuota(ctx, p.Name, p.Path, maxLeases, p.Blocked); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errCreateQuota)
		}
	default:
		return managed.ExternalUpdate{}, errors.Errorf("unknown quota type: %s", p.Type)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Quota)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotQuota)
	}

	p := cr.Spec.ForProvider

	if err := e.service.DeleteQuota(ctx, p.Type, p.Name); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteQuota)
	}

	return managed.ExternalDelete{}, nil
}
