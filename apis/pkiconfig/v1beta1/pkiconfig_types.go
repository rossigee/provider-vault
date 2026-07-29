package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const PKIConfigKind = "PKIConfig"

var PKIConfigGroupVersionKind = SchemeGroupVersion.WithKind(PKIConfigKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=pkiconfig.vault.m.crossplane.io
type PKIConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PKIConfigSpec   `json:"spec"`
	Status            PKIConfigStatus `json:"status,omitempty"`
}

type PKIConfigSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              PKIConfigParameters `json:"forProvider"`
}

type PKIConfigParameters struct {
	// Backend is the mount path of the PKI engine (e.g. "pki-production-ecdsa").
	Backend string `json:"backend"`
	// Type of CA to generate: "root_internal" or "root_exported".
	Type string `json:"type"`
	// CommonName for the root CA certificate.
	CommonName string `json:"commonName"`
	// TTL for the root CA (e.g. "87600h").
	TTL string `json:"ttl,omitempty"`
	// KeyType for the root CA key: "rsa" or "ec".
	KeyType string `json:"keyType,omitempty"`
	// KeyBits for the root CA key.
	KeyBits *int `json:"keyBits,omitempty"`
	// Organization for the root CA subject.
	Organization []string `json:"organization,omitempty"`
	// Ou for the root CA subject.
	Ou []string `json:"ou,omitempty"`
	// Country for the root CA subject.
	Country []string `json:"country,omitempty"`
	// Locality for the root CA subject.
	Locality []string `json:"locality,omitempty"`
	// Province for the root CA subject.
	Province []string `json:"province,omitempty"`
	// StreetAddress for the root CA subject.
	StreetAddress []string `json:"streetAddress,omitempty"`
	// PostalCode for the root CA subject.
	PostalCode []string `json:"postalCode,omitempty"`
	// PermittedDnsDomains for the root CA.
	PermittedDnsDomains []string `json:"permittedDnsDomains,omitempty"`
	// MaxPathLength for intermediate CA certificates.
	MaxPathLength *int `json:"maxPathLength,omitempty"`
	// ExcludeCnFromSans excludes the common name from subject alternative names.
	ExcludeCnFromSans *bool `json:"excludeCnFromSans,omitempty"`
	// IssuingCertificates URLs for CRL distribution.
	IssuingCertificates []string `json:"issuingCertificates,omitempty"`
	// CrlDistributionPoints URLs for CRL distribution.
	CrlDistributionPoints []string `json:"crlDistributionPoints,omitempty"`
	// OcspServers URLs for OCSP responders.
	OcspServers []string `json:"ocspServers,omitempty"`
}

type PKIConfigStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             PKIConfigObservation `json:"atProvider,omitempty"`
}

type PKIConfigObservation struct {
	Backend     string `json:"backend,omitempty"`
	Certificate string `json:"certificate,omitempty"`
	Serial      string `json:"serial,omitempty"`
	PrivateKey  string `json:"privateKey,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type PKIConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PKIConfig `json:"items"`
}
