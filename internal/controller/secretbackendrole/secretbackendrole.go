package secretbackendrole

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v1beta1 "github.com/rossigee/provider-vault/apis/secretbackendrole/v1beta1"
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
			kube: mgr.GetClient(),
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
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.SecretBackendRole)
	if !ok {
		return nil, errors.New(errNotSecretBackendRole)
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
	cr, ok := mg.(*v1beta1.SecretBackendRole)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSecretBackendRole)
	}

	data, err := e.service.GetSecretBackendRole(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Name = cr.Spec.ForProvider.Name

	p := cr.Spec.ForProvider
	upToDate := !clients.DriftedStringSlice(data, "allowed_domains", p.AllowedDomains)

	if clients.DriftedBool(data, "allow_subdomains", p.AllowSubdomains) {
		upToDate = false
	}
	if clients.DriftedBool(data, "allow_bare_domains", p.AllowBareDomains) {
		upToDate = false
	}
	if clients.DriftedBool(data, "allow_glob_domains", p.AllowGlobDomains) {
		upToDate = false
	}
	if clients.DriftedBool(data, "allow_any_name", p.AllowAnyName) {
		upToDate = false
	}
	if clients.DriftedString(data, "key_type", p.KeyType) {
		upToDate = false
	}
	if clients.DriftedInt(data, "key_bits", p.KeyBits) {
		upToDate = false
	}
	if clients.DriftedInt(data, "signature_bits", p.SignatureBits) {
		upToDate = false
	}
	if clients.DriftedDuration(data, "ttl", p.TTL) {
		upToDate = false
	}
	if clients.DriftedDuration(data, "max_ttl", p.MaxTTL) {
		upToDate = false
	}
	if clients.DriftedBool(data, "generate_lease", p.GenerateLease) {
		upToDate = false
	}
	if clients.DriftedBool(data, "enforce_hostnames", p.EnforceHostnames) {
		upToDate = false
	}
	if clients.DriftedBool(data, "allow_ip_sans", p.AllowIPSans) {
		upToDate = false
	}
	if clients.DriftedBool(data, "allow_localhost", p.AllowLocalhostFlag) {
		upToDate = false
	}
	if clients.DriftedBool(data, "allow_wildcard_certificates", p.AllowWildcardCertificates) {
		upToDate = false
	}
	if clients.DriftedBool(data, "server_flag", p.ServerFlag) {
		upToDate = false
	}
	if clients.DriftedBool(data, "client_flag", p.ClientFlag) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "organization", p.Organization) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "ou", p.Ou) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "country", p.Country) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "locality", p.Locality) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "province", p.Province) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "street_address", p.StreetAddress) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "postal_code", p.PostalCode) {
		upToDate = false
	}
	if clients.DriftedBool(data, "no_store", p.NoStore) {
		upToDate = false
	}
	if clients.DriftedBool(data, "require_cn", p.RequireCN) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "allowed_other_sans", p.AllowedOtherSans) {
		upToDate = false
	}
	if clients.DriftedStringSlice(data, "allowed_serial_numbers", p.AllowedSerialNumbers) {
		upToDate = false
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
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
