package gcpauthconfig

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v1beta1 "github.com/rossigee/provider-vault/apis/gcpauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotGCPAuthConfig = "managed resource is not a GCPAuthConfig custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errCreateGAC        = "cannot create GCP auth config"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.GCPAuthConfigKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.GCPAuthConfigGroupVersionKind),
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
		For(&v1beta1.GCPAuthConfig{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.GCPAuthConfig)
	if !ok {
		return nil, errors.New(errNotGCPAuthConfig)
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
	cr, ok := mg.(*v1beta1.GCPAuthConfig)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotGCPAuthConfig)
	}

	data, err := e.service.GetGCPAuthConfig(ctx, cr.Spec.ForProvider.Backend)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Backend = cr.Spec.ForProvider.Backend

	p := cr.Spec.ForProvider
	upToDate := true

	if p.ServiceAccountEmail != "" {
		if v, ok := data["service_account_email"].(string); ok && v != p.ServiceAccountEmail {
			upToDate = false
		}
	}
	if p.ProjectID != "" {
		if v, ok := data["project_id"].(string); ok && v != p.ProjectID {
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
	cr, ok := mg.(*v1beta1.GCPAuthConfig)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotGCPAuthConfig)
	}

	if err := e.configureGCPAuth(ctx, cr); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateGAC)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.GCPAuthConfig)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotGCPAuthConfig)
	}

	if err := e.configureGCPAuth(ctx, cr); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateGAC)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *external) configureGCPAuth(ctx context.Context, cr *v1beta1.GCPAuthConfig) error {
	p := cr.Spec.ForProvider
	params := make(map[string]interface{})

	if p.Credentials != "" {
		params["credentials"] = p.Credentials
	}
	if p.ServiceAccountEmail != "" {
		params["service_account_email"] = p.ServiceAccountEmail
	}
	if p.ProjectID != "" {
		params["project_id"] = p.ProjectID
	}
	if p.Zone != "" {
		params["zone"] = p.Zone
	}
	if p.ClusterName != "" {
		params["cluster_name"] = p.ClusterName
	}
	if p.MaxTTL != "" {
		params["max_ttl"] = p.MaxTTL
	}

	return e.service.ConfigureGCPAuth(ctx, p.Backend, params)
}
