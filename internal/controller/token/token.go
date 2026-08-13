package token

import (
	"context"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v1beta1 "github.com/rossigee/provider-vault/apis/token/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotToken     = "managed resource is not a Token custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errCreateToken  = "cannot create Vault token"
	errLookupToken  = "cannot lookup Vault token"
	errRenewToken   = "cannot renew Vault token"
	errRevokeToken  = "cannot revoke Vault token"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.TokenKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.TokenGroupVersionKind),
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
		For(&v1beta1.Token{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.Token)
	if !ok {
		return nil, errors.New(errNotToken)
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
	cr, ok := mg.(*v1beta1.Token)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotToken)
	}

	accessor := cr.Status.AtProvider.Accessor
	if accessor == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	tokenData, err := e.service.LookupToken(ctx, accessor)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	exp, _ := tokenData["expire_time"].(string)
	if exp != "" {
		t, parseErr := time.Parse(time.RFC3339, exp)
		if parseErr == nil {
			cr.Status.AtProvider.Expiration = t.Unix()
		}
	}

	var needsUpdate bool
	if expUnix := cr.Status.AtProvider.Expiration; expUnix > 0 {
		expiry := time.Unix(expUnix, 0)
		renewBefore := 0.5
		if cr.Spec.ForProvider.RenewBefore != nil {
			renewBefore = *cr.Spec.ForProvider.RenewBefore
		}
		remaining := time.Until(expiry)
		if remaining <= 0 {
			needsUpdate = true
		} else if cr.Spec.ForProvider.TTL != "" {
			d, parseErr := time.ParseDuration(cr.Spec.ForProvider.TTL)
			if parseErr == nil && remaining < time.Duration(float64(d)*renewBefore) {
				needsUpdate = true
			}
		}
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: !needsUpdate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Token)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotToken)
	}

	params := make(map[string]interface{})
	if cr.Spec.ForProvider.RoleName != "" {
		params["role_name"] = cr.Spec.ForProvider.RoleName
	}
	if len(cr.Spec.ForProvider.Policies) > 0 {
		params["policies"] = cr.Spec.ForProvider.Policies
	}
	if cr.Spec.ForProvider.TTL != "" {
		params["ttl"] = cr.Spec.ForProvider.TTL
	}
	if cr.Spec.ForProvider.NoParent != nil {
		params["no_parent"] = *cr.Spec.ForProvider.NoParent
	}
	if cr.Spec.ForProvider.DisplayName != "" {
		params["display_name"] = cr.Spec.ForProvider.DisplayName
	}
	if cr.Spec.ForProvider.Period != "" {
		params["period"] = cr.Spec.ForProvider.Period
	}
	if cr.Spec.ForProvider.NumUses != nil {
		params["num_uses"] = *cr.Spec.ForProvider.NumUses
	}

	data, err := e.service.CreateToken(ctx, params)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateToken)
	}

	if a, ok := data["accessor"].(string); ok {
		cr.Status.AtProvider.Accessor = a
	}
	if ct, ok := data["client_token"].(string); ok {
		cr.Status.AtProvider.ClientToken = ct
	}
	if exp, ok := data["lease_duration"].(float64); ok && exp > 0 {
		cr.Status.AtProvider.Expiration = time.Now().Unix() + int64(exp)
	}

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			"token": []byte(cr.Status.AtProvider.ClientToken),
		},
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Token)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotToken)
	}

	accessor := cr.Status.AtProvider.Accessor
	if accessor == "" {
		return managed.ExternalUpdate{}, nil
	}

	if err := e.service.RenewToken(ctx, accessor, 0); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errRenewToken)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Token)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotToken)
	}

	accessor := cr.Status.AtProvider.Accessor
	if accessor != "" {
		if err := e.service.RevokeToken(ctx, accessor); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errRevokeToken)
		}
	}

	return managed.ExternalDelete{}, nil
}
