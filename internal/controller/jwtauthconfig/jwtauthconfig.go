package jwtauthconfig

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

	v1beta1 "github.com/rossigee/provider-vault/apis/jwtauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotJWTAuthConfig = "managed resource is not a JWTAuthConfig custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errCreateJAC        = "cannot create JWT auth config"
	errDeleteJAC        = "cannot delete JWT auth config"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.JWTAuthConfigKind)

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
		resource.ManagedKind(v1beta1.JWTAuthConfigGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.JWTAuthConfig{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.JWTAuthConfig)
	if !ok {
		return nil, errors.New(errNotJWTAuthConfig)
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
	cr, ok := mg.(*v1beta1.JWTAuthConfig)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotJWTAuthConfig)
	}

	data, err := e.service.GetJWTAuthConfig(ctx, cr.Spec.ForProvider.Backend)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Backend = cr.Spec.ForProvider.Backend

	p := cr.Spec.ForProvider
	upToDate := true

	if p.OIDCDiscoveryURL != "" {
		if v, ok := data["oidc_discovery_url"].(string); ok && v != p.OIDCDiscoveryURL {
			upToDate = false
		}
	}
	if p.BoundIssuer != "" {
		if v, ok := data["bound_issuer"].(string); ok && v != p.BoundIssuer {
			upToDate = false
		}
	}
	if p.DefaultRole != "" {
		if v, ok := data["default_role"].(string); ok && v != p.DefaultRole {
			upToDate = false
		}
	}
	if p.OIDCClientID != "" {
		if v, ok := data["oidc_client_id"].(string); ok && v != p.OIDCClientID {
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
	cr, ok := mg.(*v1beta1.JWTAuthConfig)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotJWTAuthConfig)
	}

	if err := e.configureJWTAuth(ctx, cr); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateJAC)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.JWTAuthConfig)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotJWTAuthConfig)
	}

	if err := e.configureJWTAuth(ctx, cr); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateJAC)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *external) configureJWTAuth(ctx context.Context, cr *v1beta1.JWTAuthConfig) error {
	p := cr.Spec.ForProvider
	params := make(map[string]interface{})

	if p.OIDCDiscoveryURL != "" {
		params["oidc_discovery_url"] = p.OIDCDiscoveryURL
	}
	if p.OIDCDiscoveryCAPEM != "" {
		params["oidc_discovery_ca_pem"] = p.OIDCDiscoveryCAPEM
	}
	if p.OIDCClientID != "" {
		params["oidc_client_id"] = p.OIDCClientID
	}
	if p.OIDCClientSecret != "" {
		params["oidc_client_secret"] = p.OIDCClientSecret
	}
	if len(p.JWTValidationPubKeys) > 0 {
		params["jwt_validation_pubkeys"] = p.JWTValidationPubKeys
	}
	if p.BoundIssuer != "" {
		params["bound_issuer"] = p.BoundIssuer
	}
	if p.DefaultRole != "" {
		params["default_role"] = p.DefaultRole
	}
	if p.MaxTTL != "" {
		params["max_ttl"] = p.MaxTTL
	}
	if p.TTL != "" {
		params["ttl"] = p.TTL
	}
	if p.Period != "" {
		params["period"] = p.Period
	}
	if p.JWKSCacheDuration != "" {
		params["jwks_cache_duration"] = p.JWKSCacheDuration
	}

	return e.service.ConfigureJWTAuth(ctx, p.Backend, params)
}
