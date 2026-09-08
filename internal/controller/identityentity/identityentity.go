package identityentity

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/rossigee/provider-vault/internal/features"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v1beta1 "github.com/rossigee/provider-vault/apis/identityentity/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotIdentityEntity    = "managed resource is not an IdentityEntity custom resource"
	errTrackPCUsage         = "cannot track ProviderConfig usage"
	errGetPC                = "cannot get ProviderConfig"
	errGetCreds             = "cannot get credentials"
	errCreateIdentityEntity = "cannot create Vault identity entity"
	errUpdateIdentityEntity = "cannot update Vault identity entity"
	errDeleteIdentityEntity = "cannot delete Vault identity entity"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.IdentityEntityKind)

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
		resource.ManagedKind(v1beta1.IdentityEntityGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.IdentityEntity{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.IdentityEntity)
	if !ok {
		return nil, errors.New(errNotIdentityEntity)
	}

	if err := clients.TrackUsage(ctx, c.kube, cr); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	svc, err := clients.Connect(ctx, c.kube, cr)
	if err != nil {
		return nil, err
	}

	return &external{service: svc, kube: c.kube}, nil
}

type external struct {
	service *clients.VaultClient
	kube    client.Client
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.IdentityEntity)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIdentityEntity)
	}

	data, err := e.service.GetIdentityEntity(ctx, cr.Spec.ForProvider.Name)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	_ = data
	cr.Status.AtProvider.Name = cr.Spec.ForProvider.Name
	if id, ok := data["id"].(string); ok {
		cr.Status.AtProvider.ID = id
	}
	if cid, ok := data["canonical_id"].(string); ok {
		cr.Status.AtProvider.CanonicalID = cid
	}

	p := cr.Spec.ForProvider
	upToDate := !clients.DriftedBool(data, "disabled", p.Disabled)

	if clients.DriftedStringSlice(data, "policies", p.Policies) {
		upToDate = false
	}
	if clients.DriftedStringMap(data, "metadata", p.Metadata) {
		upToDate = false
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.IdentityEntity)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIdentityEntity)
	}

	p := cr.Spec.ForProvider
	params := map[string]interface{}{
		"name": p.Name,
	}
	if len(p.Policies) > 0 {
		params["policies"] = p.Policies
	}
	if len(p.Metadata) > 0 {
		params["metadata"] = p.Metadata
	}
	if p.Disabled != nil {
		params["disabled"] = *p.Disabled
	}

	data, err := e.service.CreateIdentityEntity(ctx, params)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateIdentityEntity)
	}

	if id, ok := data["id"].(string); ok {
		cr.Status.AtProvider.ID = id
	}
	cr.Status.AtProvider.Name = p.Name

	if len(p.GroupIDs) > 0 {
		log.Log.Info("group membership requires a separate IdentityGroup resource", "entity", p.Name)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.IdentityEntity)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIdentityEntity)
	}

	p := cr.Spec.ForProvider
	params := map[string]interface{}{
		"name": p.Name,
	}
	if len(p.Policies) > 0 {
		params["policies"] = p.Policies
	}
	if len(p.Metadata) > 0 {
		params["metadata"] = p.Metadata
	}
	if p.Disabled != nil {
		params["disabled"] = *p.Disabled
	}

	if err := e.service.UpdateIdentityEntity(ctx, p.Name, params); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateIdentityEntity)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.IdentityEntity)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotIdentityEntity)
	}

	if err := e.service.DeleteIdentityEntity(ctx, cr.Spec.ForProvider.Name); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteIdentityEntity)
	}

	return managed.ExternalDelete{}, nil
}
