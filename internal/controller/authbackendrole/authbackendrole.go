package authbackendrole

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/rossigee/provider-vault/internal/features"

	v1beta1 "github.com/rossigee/provider-vault/apis/authbackendrole/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotAuthBackendRole    = "managed resource is not an AuthBackendRole custom resource"
	errTrackPCUsage          = "cannot track ProviderConfig usage"
	errGetPC                 = "cannot get ProviderConfig"
	errGetCreds              = "cannot get credentials"
	errCreateAuthBackendRole = "cannot create Vault auth backend role"
	errDeleteAuthBackendRole = "cannot delete Vault auth backend role"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.AuthBackendRoleKind)

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
		resource.ManagedKind(v1beta1.AuthBackendRoleGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.AuthBackendRole{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.AuthBackendRole)
	if !ok {
		return nil, errors.New(errNotAuthBackendRole)
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
	cr, ok := mg.(*v1beta1.AuthBackendRole)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAuthBackendRole)
	}

	data, err := e.service.GetAuthBackendRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.RoleName)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.RoleName = cr.Spec.ForProvider.RoleName

	p := cr.Spec.ForProvider
	upToDate := !clients.DriftedString(data, "role_type", p.RoleType)

	if clients.DriftedStringSlice(data, "bound_audiences", p.BoundAudiences) {
		upToDate = false
	}
	if clients.DriftedString(data, "bound_subject", p.BoundSubject) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "bound_service_account_names", p.BoundServiceAccountNames) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "bound_service_account_namespaces", p.BoundServiceAccountNamespaces) {
		upToDate = false
	}
	if clients.DriftedString(data, "user_claim", p.UserClaim) {
		upToDate = false
	}
	if clients.DriftedString(data, "groups_claim", p.GroupsClaim) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "policies", p.Policies) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "token_policies", p.TokenPolicies) {
		upToDate = false
	}
	if clients.DriftedIntDuration(data, "token_ttl", p.TokenTTL) {
		upToDate = false
	}
	if clients.DriftedIntDuration(data, "token_max_ttl", p.TokenMaxTTL) {
		upToDate = false
	}
	if clients.DriftedIntDuration(data, "token_period", p.TokenPeriod) {
		upToDate = false
	}
	if clients.DriftedInt(data, "token_num_uses", p.TokenNumUses) {
		upToDate = false
	}
	if clients.DriftedString(data, "token_type", p.TokenType) {
		upToDate = false
	}
	if clients.DriftedIntDuration(data, "secret_id_ttl", p.SecretIDTTL) {
		upToDate = false
	}
	if clients.DriftedInt(data, "secret_id_num_uses", p.SecretIDNumUses) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "token_bound_cidrs", p.TokenBoundCIDRs) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "allowed_redirect_uris", p.AllowedRedirectURIs) {
		upToDate = false
	}
	if clients.DriftedInt(data, "clock_skew_leeway", p.ClockSkewLeeway) {
		upToDate = false
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.AuthBackendRole)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAuthBackendRole)
	}

	params := buildAuthBackendRoleParams(cr.Spec.ForProvider)

	if err := e.service.CreateAuthBackendRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.RoleName, params); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAuthBackendRole)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.AuthBackendRole)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAuthBackendRole)
	}

	params := buildAuthBackendRoleParams(cr.Spec.ForProvider)

	if err := e.service.CreateAuthBackendRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.RoleName, params); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateAuthBackendRole)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.AuthBackendRole)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAuthBackendRole)
	}

	if err := e.service.DeleteAuthBackendRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.RoleName); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAuthBackendRole)
	}

	return managed.ExternalDelete{}, nil
}

func buildAuthBackendRoleParams(p v1beta1.AuthBackendRoleParameters) map[string]interface{} {
	params := make(map[string]interface{})

	if p.RoleType != "" {
		params["role_type"] = p.RoleType
	}
	if len(p.BoundAudiences) > 0 {
		params["bound_audiences"] = p.BoundAudiences
	}
	if p.BoundSubject != "" {
		params["bound_subject"] = p.BoundSubject
	}
	if len(p.BoundServiceAccountNames) > 0 {
		params["bound_service_account_names"] = p.BoundServiceAccountNames
	}
	if len(p.BoundServiceAccountNamespaces) > 0 {
		params["bound_service_account_namespaces"] = p.BoundServiceAccountNamespaces
	}
	if p.UserClaim != "" {
		params["user_claim"] = p.UserClaim
	}
	if p.GroupsClaim != "" {
		params["groups_claim"] = p.GroupsClaim
	}
	if len(p.Policies) > 0 {
		params["policies"] = p.Policies
	}
	if len(p.TokenPolicies) > 0 {
		params["token_policies"] = p.TokenPolicies
	}
	if p.TokenTTL != nil {
		params["token_ttl"] = *p.TokenTTL
	}
	if p.TokenMaxTTL != nil {
		params["token_max_ttl"] = *p.TokenMaxTTL
	}
	if p.TokenPeriod != nil {
		params["token_period"] = *p.TokenPeriod
	}
	if p.TokenNumUses != nil {
		params["token_num_uses"] = *p.TokenNumUses
	}
	if p.TokenType != "" {
		params["token_type"] = p.TokenType
	}
	if p.SecretIDTTL != nil {
		params["secret_id_ttl"] = *p.SecretIDTTL
	}
	if p.SecretIDNumUses != nil {
		params["secret_id_num_uses"] = *p.SecretIDNumUses
	}
	if len(p.TokenBoundCIDRs) > 0 {
		params["token_bound_cidrs"] = p.TokenBoundCIDRs
	}
	if len(p.AllowedRedirectURIs) > 0 {
		params["allowed_redirect_uris"] = p.AllowedRedirectURIs
	}
	if p.ClockSkewLeeway != nil {
		params["clock_skew_leeway"] = *p.ClockSkewLeeway
	}

	return params
}
