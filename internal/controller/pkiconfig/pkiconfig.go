package pkiconfig

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v1beta1 "github.com/rossigee/provider-vault/apis/pkiconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotPKIConfig  = "managed resource is not a PKIConfig custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errGenerateRoot  = "cannot generate PKI root CA"
	errConfigURLs    = "cannot configure PKI URLs"
	errConfigCRL     = "cannot configure PKI CRL"
	errDeletePKIConf = "cannot delete PKI config"
)

func buildPKICRLParams(p v1beta1.PKIConfigParameters) map[string]interface{} {
	params := make(map[string]interface{})
	if p.CrlExpiry != "" {
		params["expiry"] = p.CrlExpiry
	}
	if p.CrlDisable != nil {
		params["disable"] = *p.CrlDisable
	}
	if p.OcspDisable != nil {
		params["ocsp_disable"] = *p.OcspDisable
	}
	if p.CrlAutoRebuild != nil {
		params["auto_rebuild"] = *p.CrlAutoRebuild
	}
	if p.CrlAutoRebuildGracePeriod != "" {
		params["auto_rebuild_grace_period"] = p.CrlAutoRebuildGracePeriod
	}
	if p.CrlEnableDelta != nil {
		params["enable_delta"] = *p.CrlEnableDelta
	}
	if p.CrlDeltaRebuildInterval != "" {
		params["delta_rebuild_interval"] = p.CrlDeltaRebuildInterval
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.PKIConfigKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.PKIConfigGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube: mgr.GetClient(),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()),
		managed.WithDeterministicExternalName(true),
		managed.WithManagementPolicies())

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.PKIConfig{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.PKIConfig)
	if !ok {
		return nil, errors.New(errNotPKIConfig)
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
	cr, ok := mg.(*v1beta1.PKIConfig)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPKIConfig)
	}

	ca, err := e.service.GetPKICA(ctx, cr.Spec.ForProvider.Backend)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Backend = cr.Spec.ForProvider.Backend
	cr.Status.AtProvider.Certificate = ca

	p := cr.Spec.ForProvider
	upToDate := true
	if len(p.IssuingCertificates) > 0 || len(p.CrlDistributionPoints) > 0 || len(p.OcspServers) > 0 {
		urls, err := e.service.GetPKIURLs(ctx, p.Backend)
		if err != nil {
			if clients.IsNotFound(err) {
				return managed.ExternalObservation{ResourceExists: false}, nil
			}
			return managed.ExternalObservation{}, err
		}
		if clients.DriftedStringSlice(urls, "issuing_certificates", p.IssuingCertificates) {
			upToDate = false
		}
		if clients.DriftedStringSlice(urls, "crl_distribution_points", p.CrlDistributionPoints) {
			upToDate = false
		}
		if clients.DriftedStringSlice(urls, "ocsp_servers", p.OcspServers) {
			upToDate = false
		}
	}

	if buildPKICRLParams(p) != nil {
		crl, err := e.service.GetPKICRLConfig(ctx, p.Backend)
		if err != nil {
			if clients.IsNotFound(err) {
				return managed.ExternalObservation{ResourceExists: false}, nil
			}
			return managed.ExternalObservation{}, err
		}
		if clients.DriftedDuration(crl, "expiry", p.CrlExpiry) {
			upToDate = false
		}
		if clients.DriftedBool(crl, "disable", p.CrlDisable) {
			upToDate = false
		}
		if clients.DriftedBool(crl, "ocsp_disable", p.OcspDisable) {
			upToDate = false
		}
		if clients.DriftedBool(crl, "auto_rebuild", p.CrlAutoRebuild) {
			upToDate = false
		}
		if clients.DriftedDuration(crl, "auto_rebuild_grace_period", p.CrlAutoRebuildGracePeriod) {
			upToDate = false
		}
		if clients.DriftedBool(crl, "enable_delta", p.CrlEnableDelta) {
			upToDate = false
		}
		if clients.DriftedDuration(crl, "delta_rebuild_interval", p.CrlDeltaRebuildInterval) {
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
	cr, ok := mg.(*v1beta1.PKIConfig)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPKIConfig)
	}

	p := cr.Spec.ForProvider

	params := make(map[string]interface{})
	if p.KeyType != "" {
		params["key_type"] = p.KeyType
	}
	if p.KeyBits != nil {
		params["key_bits"] = *p.KeyBits
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
	if len(p.PermittedDnsDomains) > 0 {
		params["permitted_dns_domains"] = p.PermittedDnsDomains
	}
	if p.MaxPathLength != nil {
		params["max_path_length"] = *p.MaxPathLength
	}
	if p.ExcludeCnFromSans != nil {
		params["exclude_cn_from_sans"] = *p.ExcludeCnFromSans
	}

	exportType := "internal"
	if p.Type == "root_exported" {
		exportType = "exported"
	}

	data, err := e.service.GenerateRootCA(ctx, p.Backend, exportType, p.CommonName, p.TTL, params)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGenerateRoot)
	}

	if cert, ok := data["certificate"]; ok {
		cr.Status.AtProvider.Certificate = cert.(string)
	}
	if serial, ok := data["serial_number"]; ok {
		cr.Status.AtProvider.Serial = serial.(string)
	}
	if pk, ok := data["private_key"]; ok {
		cr.Status.AtProvider.PrivateKey = pk.(string)
	}

	if len(p.IssuingCertificates) > 0 || len(p.CrlDistributionPoints) > 0 || len(p.OcspServers) > 0 {
		if err := e.service.ConfigurePKIURLs(ctx, p.Backend, p.IssuingCertificates, p.CrlDistributionPoints, p.OcspServers); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errConfigURLs)
		}
	}

	if params := buildPKICRLParams(p); params != nil {
		if err := e.service.ConfigurePKICRL(ctx, p.Backend, params); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errConfigCRL)
		}
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.PKIConfig)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPKIConfig)
	}

	p := cr.Spec.ForProvider

	if len(p.IssuingCertificates) > 0 || len(p.CrlDistributionPoints) > 0 || len(p.OcspServers) > 0 {
		if err := e.service.ConfigurePKIURLs(ctx, p.Backend, p.IssuingCertificates, p.CrlDistributionPoints, p.OcspServers); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errConfigURLs)
		}
	}

	if params := buildPKICRLParams(p); params != nil {
		if err := e.service.ConfigurePKICRL(ctx, p.Backend, params); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errConfigCRL)
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}
