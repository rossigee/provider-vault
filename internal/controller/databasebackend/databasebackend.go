package databasebackend

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/databasebackend/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
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

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.DatabaseBackendGroupVersionKind),
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
		For(&v1beta1.DatabaseBackend{}).
		Complete(r)
}

type connector struct {
	kube  client.Client
	usage resource.TrackerFn
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.DatabaseBackend)
	if !ok {
		return nil, errors.New(errNotDatabaseBackend)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &vaultv1beta1.ProviderConfig{}
	pcRef := cr.GetProviderConfigReference()

	pcName := "default"
	if pcRef != nil && pcRef.Name != "" {
		pcName = pcRef.Name
	}

	pcErr := c.kube.Get(ctx, types.NamespacedName{Name: pcName, Namespace: "crossplane-system"}, pc)
	if pcErr != nil {
		pcNamespace := cr.GetNamespace()
		if pcNamespace != "crossplane-system" {
			fallbackErr := c.kube.Get(ctx, types.NamespacedName{Name: pcName, Namespace: pcNamespace}, pc)
			if fallbackErr != nil {
				return nil, errors.Wrapf(pcErr, "cannot get ProviderConfig '%s'", pcName)
			}
		} else {
			return nil, errors.Wrapf(pcErr, "cannot get ProviderConfig '%s'", pcName)
		}
	}

	config, err := clients.GetConfig(ctx, c.kube, pc)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := clients.NewVaultClientFromConfig(config.Address, config.Token, config.Insecure)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
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

	_, err := e.service.GetDatabaseBackendConfig(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.Name = cr.Spec.ForProvider.Name
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
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
