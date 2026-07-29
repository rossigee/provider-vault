package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const CertificateKind = "Certificate"

var CertificateGroupVersionKind = SchemeGroupVersion.WithKind(CertificateKind)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=certificate.vault.m.crossplane.io
type Certificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CertificateSpec   `json:"spec"`
	Status            CertificateStatus `json:"status,omitempty"`
}

type CertificateSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              CertificateParameters `json:"forProvider"`
}

type CertificateParameters struct {
	// Backend is the mount path of the PKI engine.
	Backend string `json:"backend"`
	// Role is the name of the PKI role to issue the certificate from.
	Role string `json:"role"`
	// CommonName for the certificate.
	CommonName string `json:"commonName"`
	// AltNames is a list of subject alternative names.
	AltNames []string `json:"altNames,omitempty"`
	// IpSans is a list of IP subject alternative names.
	IpSans []string `json:"ipSans,omitempty"`
	// UriSans is a list of URI subject alternative names.
	UriSans []string `json:"uriSans,omitempty"`
	// OtherSans is a list of custom OID subject alternative names.
	OtherSans []string `json:"otherSans,omitempty"`
	// TTL for the certificate (e.g. "2160h").
	TTL string `json:"ttl,omitempty"`
	// Format of the returned certificate data: "pem", "der", "pem_bundle".
	Format string `json:"format,omitempty"`
	// PrivateKeyFormat for the private key: "der", "pkcs8".
	PrivateKeyFormat string `json:"privateKeyFormat,omitempty"`
	// RenewBefore percentage of TTL before renewal. Defaults to 0.33 (33%).
	RenewBefore *float64 `json:"renewBefore,omitempty"`
}

type CertificateStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             CertificateObservation `json:"atProvider,omitempty"`
}

type CertificateObservation struct {
	Serial       string `json:"serial,omitempty"`
	Expiration   int64  `json:"expiration,omitempty"`
	Certificate  string `json:"certificate,omitempty"`
	IssuingCa    string `json:"issuingCa,omitempty"`
	CaChain      string `json:"caChain,omitempty"`
	PrivateKey   string `json:"privateKey,omitempty"`
	PrivateKeyType string `json:"privateKeyType,omitempty"`
}

// +kubebuilder:object:root=true
type CertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Certificate `json:"items"`
}
