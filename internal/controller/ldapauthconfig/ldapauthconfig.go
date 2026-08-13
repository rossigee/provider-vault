package ldapauthconfig

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v1beta1 "github.com/rossigee/provider-vault/apis/ldapauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotLDAPAuthConfig = "managed resource is not an LDAPAuthConfig custom resource"
	errTrackPCUsage      = "cannot track ProviderConfig usage"
	errGetPC             = "cannot get ProviderConfig"
	errGetCreds          = "cannot get credentials"
	errCreateLAC         = "cannot create LDAP auth config"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.LDAPAuthConfigKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.LDAPAuthConfigGroupVersionKind),
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
		For(&v1beta1.LDAPAuthConfig{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.LDAPAuthConfig)
	if !ok {
		return nil, errors.New(errNotLDAPAuthConfig)
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
	cr, ok := mg.(*v1beta1.LDAPAuthConfig)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLDAPAuthConfig)
	}

	data, err := e.service.GetLDAPAuthConfig(ctx, cr.Spec.ForProvider.Backend)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Backend = cr.Spec.ForProvider.Backend

	p := cr.Spec.ForProvider
	upToDate := true

	if p.URL != "" {
		if v, ok := data["url"].(string); ok && v != p.URL {
			upToDate = false
		}
	}
	if p.BindDN != "" {
		if v, ok := data["binddn"].(string); ok && v != p.BindDN {
			upToDate = false
		}
	}
	if p.UserDN != "" {
		if v, ok := data["userdn"].(string); ok && v != p.UserDN {
			upToDate = false
		}
	}
	if p.UserAttr != "" {
		if v, ok := data["userattr"].(string); ok && v != p.UserAttr {
			upToDate = false
		}
	}
	if p.GroupDN != "" {
		if v, ok := data["groupdn"].(string); ok && v != p.GroupDN {
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
	cr, ok := mg.(*v1beta1.LDAPAuthConfig)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLDAPAuthConfig)
	}

	if err := e.configureLDAPAuth(ctx, cr); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateLAC)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.LDAPAuthConfig)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLDAPAuthConfig)
	}

	if err := e.configureLDAPAuth(ctx, cr); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateLAC)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *external) configureLDAPAuth(ctx context.Context, cr *v1beta1.LDAPAuthConfig) error {
	p := cr.Spec.ForProvider
	params := map[string]interface{}{
		"url":      p.URL,
		"binddn":   p.BindDN,
		"bindpass": p.BindPass,
	}
	if p.Certificate != "" {
		params["certificate"] = p.Certificate
	}
	if p.RequestTimeout != nil {
		params["request_timeout"] = *p.RequestTimeout
	}
	if p.StartTLS != nil {
		params["starttls"] = *p.StartTLS
	}
	if p.InsecureTLS != nil {
		params["insecure_tls"] = *p.InsecureTLS
	}
	if p.UserDN != "" {
		params["userdn"] = p.UserDN
	}
	if p.UserAttr != "" {
		params["userattr"] = p.UserAttr
	}
	if p.GroupDN != "" {
		params["groupdn"] = p.GroupDN
	}
	if p.GroupAttr != "" {
		params["groupattr"] = p.GroupAttr
	}
	if p.UPNDomain != "" {
		params["upndomain"] = p.UPNDomain
	}
	if p.DiscoverDN != nil {
		params["discoverdn"] = *p.DiscoverDN
	}
	if p.UserFilter != "" {
		params["userfilter"] = p.UserFilter
	}
	if p.GroupFilter != "" {
		params["groupfilter"] = p.GroupFilter
	}
	if p.MaxTTL != "" {
		params["max_ttl"] = p.MaxTTL
	}
	if p.TLSMinVersion != "" {
		params["tls_min_version"] = p.TLSMinVersion
	}
	if p.TLSMaxVersion != "" {
		params["tls_max_version"] = p.TLSMaxVersion
	}

	return e.service.ConfigureLDAPAuth(ctx, p.Backend, params)
}
