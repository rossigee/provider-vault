package awsauthconfig

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

	v1beta1 "github.com/rossigee/provider-vault/apis/awsauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotAWSAuthConfig = "managed resource is not an AWSAuthConfig custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errCreateAAC        = "cannot create AWS auth config"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.AWSAuthConfigKind)

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
		resource.ManagedKind(v1beta1.AWSAuthConfigGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.AWSAuthConfig{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.AWSAuthConfig)
	if !ok {
		return nil, errors.New(errNotAWSAuthConfig)
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
	cr, ok := mg.(*v1beta1.AWSAuthConfig)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAWSAuthConfig)
	}

	data, err := e.service.GetAWSAuthConfig(ctx, cr.Spec.ForProvider.Backend)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Backend = cr.Spec.ForProvider.Backend

	p := cr.Spec.ForProvider
	upToDate := true

	if p.IAMServerIDHeaderValue != "" {
		if v, ok := data["iam_server_id_header_value"].(string); ok && v != p.IAMServerIDHeaderValue {
			upToDate = false
		}
	}
	if p.STSEndpoint != "" {
		if v, ok := data["sts_endpoint"].(string); ok && v != p.STSEndpoint {
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
	cr, ok := mg.(*v1beta1.AWSAuthConfig)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAWSAuthConfig)
	}

	if err := e.configureAWSAuth(ctx, cr); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAAC)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.AWSAuthConfig)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAWSAuthConfig)
	}

	if err := e.configureAWSAuth(ctx, cr); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateAAC)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *external) configureAWSAuth(ctx context.Context, cr *v1beta1.AWSAuthConfig) error {
	p := cr.Spec.ForProvider
	params := make(map[string]interface{})

	if p.IAMServerIDHeaderValue != "" {
		params["iam_server_id_header_value"] = p.IAMServerIDHeaderValue
	}
	if p.IAMRequestURL != "" {
		params["iam_request_url"] = p.IAMRequestURL
	}
	if p.IAMRequestPayload != "" {
		params["iam_request_payload"] = p.IAMRequestPayload
	}
	if p.STSEndpoint != "" {
		params["sts_endpoint"] = p.STSEndpoint
	}
	if p.STSFallbackRegion != "" {
		params["sts_fallback_region"] = p.STSFallbackRegion
	}
	if p.STSDisableRedirect != nil {
		params["sts_disable_redirect"] = *p.STSDisableRedirect
	}
	if p.MaxTTL != "" {
		params["max_ttl"] = p.MaxTTL
	}
	if p.AccessIdentity != "" {
		params["access_identity"] = p.AccessIdentity
	}

	return e.service.ConfigureAWSAuth(ctx, p.Backend, params)
}
