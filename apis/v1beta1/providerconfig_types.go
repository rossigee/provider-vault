package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=vault.m.crossplane.io
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProviderConfigSpec   `json:"spec"`
	Status            ProviderConfigStatus `json:"status,omitempty"`
}

type ProviderConfigSpec struct {
	Credentials        ProviderCredentials  `json:"credentials"`
	Address            string              `json:"address"`
	InsecureSkipVerify *bool               `json:"insecureSkipVerify,omitempty"`
	TLS                *TLSConfig          `json:"tls,omitempty"`
	// VaultNamespace is an optional Vault Enterprise namespace to use for all
	// API requests. If set, it will be sent as the X-Vault-Namespace header.
	VaultNamespace *string `json:"vaultNamespace,omitempty"`
}

type TLSConfig struct {
	// CACertSecretRef references a secret key containing the CA certificate
	// (PEM-encoded) to use when verifying the Vault server's certificate.
	CACertSecretRef *xpv1.SecretKeySelector `json:"caCertSecretRef,omitempty"`
	// ClientCertSecretRef references a secret key containing the TLS client
	// certificate (PEM-encoded) and its corresponding private key for
	// mutual TLS authentication with Vault. The referenced secret must have
	// keys "tls.crt" and "tls.key".
	ClientCertSecretRef *xpv1.SecretKeySelector `json:"clientCertSecretRef,omitempty"`
}

type ProviderCredentials struct {
	Source xpv1.CredentialsSource `json:"source"`
	xpv1.CommonCredentialSelectors `json:",inline"`
}

type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
type ProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	xpv1.ProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true
type ProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfigUsage `json:"items"`
}

// +kubebuilder:object:root=true
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}
