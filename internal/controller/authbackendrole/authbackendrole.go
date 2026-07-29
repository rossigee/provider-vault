package authbackendrole

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/authbackendrole/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
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

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.AuthBackendRoleGroupVersionKind),
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
		For(&v1beta1.AuthBackendRole{}).
		Complete(r)
}

type connector struct {
	kube  client.Client
	usage resource.TrackerFn
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.AuthBackendRole)
	if !ok {
		return nil, errors.New(errNotAuthBackendRole)
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
	cr, ok := mg.(*v1beta1.AuthBackendRole)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAuthBackendRole)
	}

	_, err := e.service.GetAuthBackendRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.RoleName)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.RoleName = cr.Spec.ForProvider.RoleName
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
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
