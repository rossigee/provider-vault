package certificate

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/certificate/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotCertificate    = "managed resource is not a Certificate custom resource"
	errTrackPCUsage      = "cannot track ProviderConfig usage"
	errGetPC             = "cannot get ProviderConfig"
	errGetCreds          = "cannot get credentials"
	errIssueCert         = "cannot issue certificate"
	errRevokeCert        = "cannot revoke certificate"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.CertificateKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.CertificateGroupVersionKind),
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
		For(&v1beta1.Certificate{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.Certificate)
	if !ok {
		return nil, errors.New(errNotCertificate)
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
	cr, ok := mg.(*v1beta1.Certificate)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCertificate)
	}

	serial := cr.Status.AtProvider.Serial
	if serial == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	certData, err := e.service.GetPKICert(ctx, cr.Spec.ForProvider.Backend, serial)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	certStr, _ := certData["certificate"].(string)
	cr.Status.AtProvider.Certificate = certStr

	var needsUpdate bool
	if exp, ok := certData["expiration"]; ok {
		expFloat, _ := exp.(float64)
		cr.Status.AtProvider.Expiration = int64(expFloat)

		expiry := time.Unix(int64(expFloat), 0)
		renewBefore := 0.33
		if cr.Spec.ForProvider.RenewBefore != nil {
			renewBefore = *cr.Spec.ForProvider.RenewBefore
		}

		remaining := time.Until(expiry)
		if remaining <= 0 {
			needsUpdate = true
		} else if ttl := cr.Spec.ForProvider.TTL; ttl != "" {
			d, parseErr := time.ParseDuration(ttl)
			if parseErr == nil && remaining < time.Duration(float64(d)*renewBefore) {
				needsUpdate = true
			}
		}
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: !needsUpdate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Certificate)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCertificate)
	}

	data, err := e.issueCertificate(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	updateStatus(cr, data)
	return managed.ExternalCreation{
		ConnectionDetails: toConnectionDetails(data),
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Certificate)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCertificate)
	}

	if cr.Spec.ForProvider.Backend == "" || cr.Spec.ForProvider.Role == "" {
		return managed.ExternalUpdate{}, nil
	}

	serial := cr.Status.AtProvider.Serial
	if serial != "" {
		_ = e.service.RevokeCertificate(ctx, cr.Spec.ForProvider.Backend, serial)
	}

	data, err := e.issueCertificate(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	updateStatus(cr, data)
	return managed.ExternalUpdate{
		ConnectionDetails: toConnectionDetails(data),
	}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Certificate)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotCertificate)
	}

	serial := cr.Status.AtProvider.Serial
	if serial != "" {
		if err := e.service.RevokeCertificate(ctx, cr.Spec.ForProvider.Backend, serial); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errRevokeCert)
		}
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) issueCertificate(ctx context.Context, cr *v1beta1.Certificate) (map[string]interface{}, error) {
	p := cr.Spec.ForProvider

	params := make(map[string]interface{})
	if len(p.AltNames) > 0 {
		params["alt_names"] = p.AltNames
	}
	if len(p.IpSans) > 0 {
		params["ip_sans"] = p.IpSans
	}
	if len(p.UriSans) > 0 {
		params["uri_sans"] = p.UriSans
	}
	if len(p.OtherSans) > 0 {
		params["other_sans"] = p.OtherSans
	}
	if p.TTL != "" {
		params["ttl"] = p.TTL
	}
	if p.Format != "" {
		params["format"] = p.Format
	}
	if p.PrivateKeyFormat != "" {
		params["private_key_format"] = p.PrivateKeyFormat
	}

	data, err := e.service.IssueCertificate(ctx, p.Backend, p.Role, p.CommonName, params)
	if err != nil {
		return nil, errors.Wrap(err, errIssueCert)
	}

	return data, nil
}

func updateStatus(cr *v1beta1.Certificate, data map[string]interface{}) {
	if s, ok := data["serial_number"].(string); ok {
		cr.Status.AtProvider.Serial = s
	}
	if cert, ok := data["certificate"].(string); ok {
		cr.Status.AtProvider.Certificate = cert
	}
	if ca, ok := data["issuing_ca"].(string); ok {
		cr.Status.AtProvider.IssuingCa = ca
	}
	if chain, ok := data["ca_chain"].([]interface{}); ok {
		for _, c := range chain {
			if cr.Status.AtProvider.CaChain != "" {
				cr.Status.AtProvider.CaChain += "\n"
			}
			cr.Status.AtProvider.CaChain += fmt.Sprint(c)
		}
	}
	if pk, ok := data["private_key"].(string); ok {
		cr.Status.AtProvider.PrivateKey = pk
	}
	if pkt, ok := data["private_key_type"].(string); ok {
		cr.Status.AtProvider.PrivateKeyType = pkt
	}
}

func toConnectionDetails(data map[string]interface{}) managed.ConnectionDetails {
	cd := managed.ConnectionDetails{}
	if cert, ok := data["certificate"].(string); ok {
		cd["tls.crt"] = []byte(cert)
	}
	if ca, ok := data["issuing_ca"].(string); ok {
		cd["ca.crt"] = []byte(ca)
	}
	if pk, ok := data["private_key"].(string); ok {
		cd["tls.key"] = []byte(pk)
	}
	return cd
}
