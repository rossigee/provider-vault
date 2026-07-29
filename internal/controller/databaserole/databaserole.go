package databaserole

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/databaserole/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
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

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.DatabaseRoleGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error { return nil }),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()),
		managed.WithDeterministicExternalName(true))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.DatabaseRole{}).
		Complete(r)
}

type connector struct {
	kube  client.Client
	usage resource.TrackerFn
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.DatabaseRole)
	if !ok {
		return nil, errors.New(errNotDatabaseRole)
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

	svc, err := config.NewClient()
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
	cr, ok := mg.(*v1beta1.DatabaseRole)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDatabaseRole)
	}

	_, err := e.service.GetDatabaseRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name)
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
