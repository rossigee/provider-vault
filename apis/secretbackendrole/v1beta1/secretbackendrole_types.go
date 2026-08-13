package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const SecretBackendRoleKind = "SecretBackendRole"

var SecretBackendRoleGroupVersionKind = SchemeGroupVersion.WithKind(SecretBackendRoleKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=secretbackendrole.vault.m.crossplane.io
type SecretBackendRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SecretBackendRoleSpec   `json:"spec"`
	Status            SecretBackendRoleStatus `json:"status,omitempty"`
}

type SecretBackendRoleSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              SecretBackendRoleParameters `json:"forProvider"`
}

type SecretBackendRoleParameters struct {
	Backend                    string   `json:"backend"`
	Name                       string   `json:"name"`
	AllowedDomains             []string `json:"allowedDomains,omitempty"`
	AllowSubdomains            *bool    `json:"allowSubdomains,omitempty"`
	AllowBareDomains           *bool    `json:"allowBareDomains,omitempty"`
	AllowGlobDomains           *bool    `json:"allowGlobDomains,omitempty"`
	AllowAnyName               *bool    `json:"allowAnyName,omitempty"`
	KeyType                    string   `json:"keyType,omitempty"`
	KeyBits                    *int     `json:"keyBits,omitempty"`
	SignatureBits              *int     `json:"signatureBits,omitempty"`
	TTL                        string   `json:"ttl,omitempty"`
	MaxTTL                     string   `json:"maxTtl,omitempty"`
	GenerateLease              *bool    `json:"generateLease,omitempty"`
	EnforceHostnames           *bool    `json:"enforceHostnames,omitempty"`
	AllowIPSans                *bool    `json:"allowIpSans,omitempty"`
	AllowLocalhostFlag         *bool    `json:"allowLocalhostFlag,omitempty"`
	AllowWildcardCertificates  *bool    `json:"allowWildcardCertificates,omitempty"`
	ServerFlag                 *bool    `json:"serverFlag,omitempty"`
	ClientFlag                 *bool    `json:"clientFlag,omitempty"`
	Organization               []string `json:"organization,omitempty"`
	Ou                         []string `json:"ou,omitempty"`
	Country                    []string `json:"country,omitempty"`
	Locality                   []string `json:"locality,omitempty"`
	Province                   []string `json:"province,omitempty"`
	StreetAddress              []string `json:"streetAddress,omitempty"`
	PostalCode                 []string `json:"postalCode,omitempty"`
	NoStore                    *bool    `json:"noStore,omitempty"`
	RequireCN                  *bool    `json:"requireCn,omitempty"`
	AllowedOtherSans           []string `json:"allowedOtherSans,omitempty"`
	AllowedSerialNumbers       []string `json:"allowedSerialNumbers,omitempty"`
	KeyUsage                   []string `json:"keyUsage,omitempty"`
	ExtKeyUsage                []string `json:"extKeyUsage,omitempty"`
}

type SecretBackendRoleStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             SecretBackendRoleObservation `json:"atProvider,omitempty"`
}

type SecretBackendRoleObservation struct {
	Name string `json:"name,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type SecretBackendRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretBackendRole `json:"items"`
}
