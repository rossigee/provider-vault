package databaserole

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

	v1beta1 "github.com/rossigee/provider-vault/apis/databaserole/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotDatabaseRole    = "managed resource is not a DatabaseRole custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"
	errCreateDatabaseRole = "cannot create Vault database role"
	errDeleteDatabaseRole = "cannot delete Vault database role"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.DatabaseRoleKind)

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
		resource.ManagedKind(v1beta1.DatabaseRoleGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.DatabaseRole{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.DatabaseRole)
	if !ok {
		return nil, errors.New(errNotDatabaseRole)
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
	cr, ok := mg.(*v1beta1.DatabaseRole)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDatabaseRole)
	}

	data, err := e.service.GetDatabaseRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Name = cr.Spec.ForProvider.Name

	p := cr.Spec.ForProvider
	upToDate := !clients.DriftedString(data, "db_name", p.DBName)

	if clients.DriftedStringSlice(data, "creation_statements", p.CreationStatements) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "revocation_statements", p.RevocationStatements) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "rollback_statements", p.RollbackStatements) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "renew_statements", p.RenewStatements) {
		upToDate = false
	}
	if clients.DriftedDuration(data, "default_ttl", p.DefaultTTL) {
		upToDate = false
	}
	if clients.DriftedDuration(data, "max_ttl", p.MaxTTL) {
		upToDate = false
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.DatabaseRole)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDatabaseRole)
	}

	params := map[string]interface{}{
		"db_name": cr.Spec.ForProvider.DBName,
	}
	if len(cr.Spec.ForProvider.CreationStatements) > 0 {
		params["creation_statements"] = cr.Spec.ForProvider.CreationStatements
	}
	if len(cr.Spec.ForProvider.RevocationStatements) > 0 {
		params["revocation_statements"] = cr.Spec.ForProvider.RevocationStatements
	}
	if len(cr.Spec.ForProvider.RollbackStatements) > 0 {
		params["rollback_statements"] = cr.Spec.ForProvider.RollbackStatements
	}
	if len(cr.Spec.ForProvider.RenewStatements) > 0 {
		params["renew_statements"] = cr.Spec.ForProvider.RenewStatements
	}
	if cr.Spec.ForProvider.DefaultTTL != "" {
		params["default_ttl"] = cr.Spec.ForProvider.DefaultTTL
	}
	if cr.Spec.ForProvider.MaxTTL != "" {
		params["max_ttl"] = cr.Spec.ForProvider.MaxTTL
	}

	if err := e.service.CreateDatabaseRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name, params); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateDatabaseRole)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.DatabaseRole)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDatabaseRole)
	}

	params := map[string]interface{}{
		"db_name": cr.Spec.ForProvider.DBName,
	}
	if len(cr.Spec.ForProvider.CreationStatements) > 0 {
		params["creation_statements"] = cr.Spec.ForProvider.CreationStatements
	}
	if len(cr.Spec.ForProvider.RevocationStatements) > 0 {
		params["revocation_statements"] = cr.Spec.ForProvider.RevocationStatements
	}
	if len(cr.Spec.ForProvider.RollbackStatements) > 0 {
		params["rollback_statements"] = cr.Spec.ForProvider.RollbackStatements
	}
	if len(cr.Spec.ForProvider.RenewStatements) > 0 {
		params["renew_statements"] = cr.Spec.ForProvider.RenewStatements
	}
	if cr.Spec.ForProvider.DefaultTTL != "" {
		params["default_ttl"] = cr.Spec.ForProvider.DefaultTTL
	}
	if cr.Spec.ForProvider.MaxTTL != "" {
		params["max_ttl"] = cr.Spec.ForProvider.MaxTTL
	}

	if err := e.service.CreateDatabaseRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name, params); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateDatabaseRole)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.DatabaseRole)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotDatabaseRole)
	}

	if err := e.service.DeleteDatabaseRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteDatabaseRole)
	}

	return managed.ExternalDelete{}, nil
}
