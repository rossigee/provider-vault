package azureauthconfig

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v1beta1 "github.com/rossigee/provider-vault/apis/azureauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotAzureAuthConfig = "managed resource is not an AzureAuthConfig custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"
	errCreateAzAC         = "cannot create Azure auth config"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.AzureAuthConfigKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.AzureAuthConfigGroupVersionKind),
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
		For(&v1beta1.AzureAuthConfig{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.AzureAuthConfig)
	if !ok {
		return nil, errors.New(errNotAzureAuthConfig)
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
	cr, ok := mg.(*v1beta1.AzureAuthConfig)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAzureAuthConfig)
	}

	data, err := e.service.GetAzureAuthConfig(ctx, cr.Spec.ForProvider.Backend)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Backend = cr.Spec.ForProvider.Backend

	p := cr.Spec.ForProvider
	upToDate := true

	if p.TenantID != "" {
		if v, ok := data["tenant_id"].(string); ok && v != p.TenantID {
			upToDate = false
		}
	}
	if p.ClientID != "" {
		if v, ok := data["client_id"].(string); ok && v != p.ClientID {
			upToDate = false
		}
	}
	if p.Resource != "" {
		if v, ok := data["resource"].(string); ok && v != p.Resource {
			upToDate = false
		}
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.AzureAuthConfig)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAzureAuthConfig)
	}

	if err := e.configureAzureAuth(ctx, cr); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAzAC)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.AzureAuthConfig)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAzureAuthConfig)
	}

	if err := e.configureAzureAuth(ctx, cr); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateAzAC)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *external) configureAzureAuth(ctx context.Context, cr *v1beta1.AzureAuthConfig) error {
	p := cr.Spec.ForProvider
	params := map[string]interface{}{
		"tenant_id": p.TenantID,
	}
	if p.Resource != "" {
		params["resource"] = p.Resource
	}
	if p.ClientID != "" {
		params["client_id"] = p.ClientID
	}
	if p.ClientSecret != "" {
		params["client_secret"] = p.ClientSecret
	}
	if p.Environment != "" {
		params["environment"] = p.Environment
	}
	if p.LoginMaxUserRetry != nil {
		params["login_max_user_retry"] = *p.LoginMaxUserRetry
	}
	if len(p.LoginScopes) > 0 {
		params["login_scopes"] = p.LoginScopes
	}

	return e.service.ConfigureAzureAuth(ctx, p.Backend, params)
}
