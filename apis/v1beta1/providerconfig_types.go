package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
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
