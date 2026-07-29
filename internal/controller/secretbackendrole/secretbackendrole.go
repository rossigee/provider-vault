package secretbackendrole

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/secretbackendrole/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotSecretBackendRole    = "managed resource is not a SecretBackendRole custom resource"
	errTrackPCUsage            = "cannot track ProviderConfig usage"
	errGetPC                   = "cannot get ProviderConfig"
	errGetCreds                = "cannot get credentials"
	errCreateSecretBackendRole = "cannot create Vault secret backend role"
	errDeleteSecretBackendRole = "cannot delete Vault secret backend role"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.SecretBackendRoleKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.SecretBackendRoleGroupVersionKind),
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
		For(&v1beta1.SecretBackendRole{}).
		Complete(r)
}

type connector struct {
	kube  client.Client
	usage resource.TrackerFn
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.SecretBackendRole)
	if !ok {
		return nil, errors.New(errNotSecretBackendRole)
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
	cr, ok := mg.(*v1beta1.SecretBackendRole)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSecretBackendRole)
	}

	_, err := e.service.GetSecretBackendRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name)
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
	cr, ok := mg.(*v1beta1.SecretBackendRole)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotSecretBackendRole)
	}

	params := buildSecretBackendRoleParams(cr.Spec.ForProvider)

	if err := e.service.CreateSecretBackendRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name, params); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateSecretBackendRole)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.SecretBackendRole)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotSecretBackendRole)
	}

	params := buildSecretBackendRoleParams(cr.Spec.ForProvider)

	if err := e.service.CreateSecretBackendRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name, params); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateSecretBackendRole)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.SecretBackendRole)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotSecretBackendRole)
	}

	if err := e.service.DeleteSecretBackendRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteSecretBackendRole)
	}

	return managed.ExternalDelete{}, nil
}

func buildSecretBackendRoleParams(p v1beta1.SecretBackendRoleParameters) map[string]interface{} {
	params := make(map[string]interface{})

	if len(p.AllowedDomains) > 0 {
		params["allowed_domains"] = p.AllowedDomains
	}
	if p.AllowSubdomains != nil {
		params["allow_subdomains"] = *p.AllowSubdomains
	}
	if p.AllowBareDomains != nil {
		params["allow_bare_domains"] = *p.AllowBareDomains
	}
	if p.AllowGlobDomains != nil {
		params["allow_glob_domains"] = *p.AllowGlobDomains
	}
	if p.AllowAnyName != nil {
		params["allow_any_name"] = *p.AllowAnyName
	}
	if p.KeyType != "" {
		params["key_type"] = p.KeyType
	}
	if p.KeyBits != nil {
		params["key_bits"] = *p.KeyBits
	}
	if p.SignatureBits != nil {
		params["signature_bits"] = *p.SignatureBits
	}
	if p.TTL != "" {
		params["ttl"] = p.TTL
	}
	if p.MaxTTL != "" {
		params["max_ttl"] = p.MaxTTL
	}
	if p.GenerateLease != nil {
		params["generate_lease"] = *p.GenerateLease
	}
	if p.EnforceHostnames != nil {
		params["enforce_hostnames"] = *p.EnforceHostnames
	}
	if p.AllowIPSans != nil {
		params["allow_ip_sans"] = *p.AllowIPSans
	}
	if p.AllowLocalhostFlag != nil {
		params["allow_localhost"] = *p.AllowLocalhostFlag
	}
	if p.AllowWildcardCertificates != nil {
		params["allow_wildcard_certificates"] = *p.AllowWildcardCertificates
	}
	if p.ServerFlag != nil {
		params["server_flag"] = *p.ServerFlag
	}
	if p.ClientFlag != nil {
		params["client_flag"] = *p.ClientFlag
	}
	if len(p.Organization) > 0 {
		params["organization"] = p.Organization
	}
	if len(p.Ou) > 0 {
		params["ou"] = p.Ou
	}
	if len(p.Country) > 0 {
		params["country"] = p.Country
	}
	if len(p.Locality) > 0 {
		params["locality"] = p.Locality
	}
	if len(p.Province) > 0 {
		params["province"] = p.Province
	}
	if len(p.StreetAddress) > 0 {
		params["street_address"] = p.StreetAddress
	}
	if len(p.PostalCode) > 0 {
		params["postal_code"] = p.PostalCode
	}
	if p.NoStore != nil {
		params["no_store"] = *p.NoStore
	}
	if p.RequireCN != nil {
		params["require_cn"] = *p.RequireCN
	}
	if len(p.AllowedOtherSans) > 0 {
		params["allowed_other_sans"] = p.AllowedOtherSans
	}
	if len(p.AllowedSerialNumbers) > 0 {
		params["allowed_serial_numbers"] = p.AllowedSerialNumbers
	}

	return params
}
