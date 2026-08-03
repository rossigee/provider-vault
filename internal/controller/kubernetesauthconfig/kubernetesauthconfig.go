package kubernetesauthconfig

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/kubernetesauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotKubernetesAuthConfig = "managed resource is not a KubernetesAuthConfig custom resource"
	errTrackPCUsage            = "cannot track ProviderConfig usage"
	errGetPC                   = "cannot get ProviderConfig"
	errGetCreds                = "cannot get credentials"
	errCreateKAC               = "cannot create Kubernetes auth config"
	errLookupKAC               = "cannot lookup Kubernetes auth config"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.KubernetesAuthConfigKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.KubernetesAuthConfigGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()),
		managed.WithDeterministicExternalName(true))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.KubernetesAuthConfig{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.KubernetesAuthConfig)
	if !ok {
		return nil, errors.New(errNotKubernetesAuthConfig)
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
	cr, ok := mg.(*v1beta1.KubernetesAuthConfig)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotKubernetesAuthConfig)
	}

	backend := cr.Spec.ForProvider.Backend
	observed, err := e.service.GetKubernetesAuthConfig(ctx, backend)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider = v1beta1.KubernetesAuthConfigObservation{
		Backend: backend,
	}

	// Check if spec matches observed state
	observedHost, _ := observed["kubernetes_host"].(string)
	upToDate := observedHost == cr.Spec.ForProvider.KubernetesHost

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.KubernetesAuthConfig)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotKubernetesAuthConfig)
	}

	params := cr.Spec.ForProvider
	vaultParams := map[string]interface{}{
		"kubernetes_host": params.KubernetesHost,
	}
	if params.KubernetesCACert != "" {
		vaultParams["kubernetes_ca_cert"] = params.KubernetesCACert
	}
	if params.TokenReviewerJWT != "" {
		vaultParams["token_reviewer_jwt"] = params.TokenReviewerJWT
	}
	if params.Issuer != "" {
		vaultParams["issuer"] = params.Issuer
	}
	if params.DisableISSValidation != nil {
		vaultParams["disable_iss_validation"] = *params.DisableISSValidation
	}
	if params.DisableLocalCAJWT != nil {
		vaultParams["disable_local_ca_jwt"] = *params.DisableLocalCAJWT
	}

	if err := e.service.ConfigureKubernetesAuth(ctx, params.Backend, vaultParams); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateKAC)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.KubernetesAuthConfig)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotKubernetesAuthConfig)
	}

	params := cr.Spec.ForProvider
	vaultParams := map[string]interface{}{
		"kubernetes_host": params.KubernetesHost,
	}
	if params.KubernetesCACert != "" {
		vaultParams["kubernetes_ca_cert"] = params.KubernetesCACert
	}
	if params.TokenReviewerJWT != "" {
		vaultParams["token_reviewer_jwt"] = params.TokenReviewerJWT
	}
	if params.Issuer != "" {
		vaultParams["issuer"] = params.Issuer
	}
	if params.DisableISSValidation != nil {
		vaultParams["disable_iss_validation"] = *params.DisableISSValidation
	}
	if params.DisableLocalCAJWT != nil {
		vaultParams["disable_local_ca_jwt"] = *params.DisableLocalCAJWT
	}

	if err := e.service.ConfigureKubernetesAuth(ctx, params.Backend, vaultParams); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateKAC)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	// No delete API for Kubernetes auth config; config is removed with the mount.
	return managed.ExternalDelete{}, nil
}
