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
	xpv1.ProviderConfigSpec `json:",inline"`
	Address                 string              `json:"address"`
	TokenSecretRef          xpv1.SecretReference `json:"tokenSecretRef"`
	InsecureSkipVerify      *bool               `json:"insecureSkipVerify,omitempty"`
}

// +kubebuilder:object:root=true
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
	Spec              ProviderConfigUsageSpec `json:"spec"`
}

type ProviderConfigUsageSpec struct {
	xpv1.ProviderConfigUsageSpec `json:",inline"`
}

func (p *ProviderConfig) GetProviderConfigSpec() xpv1.ProviderConfigSpec {
	return p.Spec.ProviderConfigSpec
}
