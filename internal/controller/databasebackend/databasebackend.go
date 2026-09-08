package databasebackend

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

	v1beta1 "github.com/rossigee/provider-vault/apis/databasebackend/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotDatabaseBackend    = "managed resource is not a DatabaseBackend custom resource"
	errTrackPCUsage          = "cannot track ProviderConfig usage"
	errGetPC                 = "cannot get ProviderConfig"
	errGetCreds              = "cannot get credentials"
	errCreateDatabaseBackend = "cannot create database backend config"
	errDeleteDatabaseBackend = "cannot delete database backend config"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.DatabaseBackendKind)

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
		resource.ManagedKind(v1beta1.DatabaseBackendGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.DatabaseBackend{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.DatabaseBackend)
	if !ok {
		return nil, errors.New(errNotDatabaseBackend)
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
	cr, ok := mg.(*v1beta1.DatabaseBackend)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDatabaseBackend)
	}

	data, err := e.service.GetDatabaseBackendConfig(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Name = cr.Spec.ForProvider.Name

	p := cr.Spec.ForProvider
	upToDate := !clients.DriftedString(data, "connection_url", p.ConnectionURL)

	if clients.DriftedString(data, "username", p.Username) {
		upToDate = false
	}
	if clients.DriftedString(data, "plugin_name", p.PluginName) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "allowed_roles", p.AllowedRoles) {
		upToDate = false
	}
	if clients.DriftedDuration(data, "max_connection_lifetime", p.MaxConnectionLifetime) {
		upToDate = false
	}
	if clients.DriftedInt(data, "max_idle_connections", p.MaxIdleConnections) {
		upToDate = false
	}
	if clients.DriftedInt(data, "max_open_connections", p.MaxOpenConnections) {
		upToDate = false
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.DatabaseBackend)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDatabaseBackend)
	}

	p := cr.Spec.ForProvider
	params := map[string]interface{}{
		"connection_url": p.ConnectionURL,
		"username":       p.Username,
		"password":       p.Password,
	}
	if p.PluginName != "" {
		params["plugin_name"] = p.PluginName
	}
	if len(p.AllowedRoles) > 0 {
		params["allowed_roles"] = p.AllowedRoles
	}
	if p.MaxConnectionLifetime != "" {
		params["max_connection_lifetime"] = p.MaxConnectionLifetime
	}
	if p.MaxIdleConnections != nil {
		params["max_idle_connections"] = *p.MaxIdleConnections
	}
	if p.MaxOpenConnections != nil {
		params["max_open_connections"] = *p.MaxOpenConnections
	}
	if p.VerifyConnection != nil {
		params["verify_connection"] = *p.VerifyConnection
	}

	if err := e.service.CreateDatabaseBackendConfig(ctx, p.Backend, p.Name, params); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateDatabaseBackend)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.DatabaseBackend)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDatabaseBackend)
	}

	p := cr.Spec.ForProvider
	params := map[string]interface{}{
		"connection_url": p.ConnectionURL,
		"username":       p.Username,
		"password":       p.Password,
	}
	if p.PluginName != "" {
		params["plugin_name"] = p.PluginName
	}
	if len(p.AllowedRoles) > 0 {
		params["allowed_roles"] = p.AllowedRoles
	}
	if p.MaxConnectionLifetime != "" {
		params["max_connection_lifetime"] = p.MaxConnectionLifetime
	}
	if p.MaxIdleConnections != nil {
		params["max_idle_connections"] = *p.MaxIdleConnections
	}
	if p.MaxOpenConnections != nil {
		params["max_open_connections"] = *p.MaxOpenConnections
	}

	if err := e.service.CreateDatabaseBackendConfig(ctx, p.Backend, p.Name, params); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateDatabaseBackend)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.DatabaseBackend)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotDatabaseBackend)
	}

	if err := e.service.DeleteDatabaseBackendConfig(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteDatabaseBackend)
	}

	return managed.ExternalDelete{}, nil
}
