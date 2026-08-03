package mount

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/mount/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotMount    = "managed resource is not a Mount custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errCreateMount  = "cannot create Vault mount"
	errDeleteMount  = "cannot delete Vault mount"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.MountKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.MountGroupVersionKind),
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
		For(&v1beta1.Mount{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.Mount)
	if !ok {
		return nil, errors.New(errNotMount)
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
	cr, ok := mg.(*v1beta1.Mount)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotMount)
	}

	data, err := e.service.GetMount(ctx, cr.Spec.ForProvider.Path)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Path = cr.Spec.ForProvider.Path

	upToDate := true
	if d, ok := data["description"].(string); ok && cr.Spec.ForProvider.Description != "" && d != cr.Spec.ForProvider.Description {
		upToDate = false
	}
	if cfg, ok := data["config"].(map[string]interface{}); ok {
		if d, ok := cfg["default_lease_ttl"].(float64); ok && cr.Spec.ForProvider.DefaultLeaseTTL > 0 && int64(d) != int64(cr.Spec.ForProvider.DefaultLeaseTTL) {
			upToDate = false
		}
		if d, ok := cfg["max_lease_ttl"].(float64); ok && cr.Spec.ForProvider.MaxLeaseTTL > 0 && int64(d) != int64(cr.Spec.ForProvider.MaxLeaseTTL) {
			upToDate = false
		}
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Mount)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotMount)
	}

	if err := e.service.EnableMount(ctx,
		cr.Spec.ForProvider.Path,
		cr.Spec.ForProvider.Type,
		cr.Spec.ForProvider.Description,
		cr.Spec.ForProvider.DefaultLeaseTTL,
		cr.Spec.ForProvider.MaxLeaseTTL,
		cr.Spec.ForProvider.Options,
		cr.Spec.ForProvider.Config,
	); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateMount)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Mount)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotMount)
	}

	if err := e.service.EnableMount(ctx,
		cr.Spec.ForProvider.Path,
		cr.Spec.ForProvider.Type,
		cr.Spec.ForProvider.Description,
		cr.Spec.ForProvider.DefaultLeaseTTL,
		cr.Spec.ForProvider.MaxLeaseTTL,
		cr.Spec.ForProvider.Options,
		cr.Spec.ForProvider.Config,
	); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateMount)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Mount)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotMount)
	}

	if err := e.service.DisableMount(ctx, cr.Spec.ForProvider.Path); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteMount)
	}

	return managed.ExternalDelete{}, nil
}
