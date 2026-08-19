package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const VaultNamespaceKind = "VaultNamespace"

var VaultNamespaceGroupVersionKind = SchemeGroupVersion.WithKind(VaultNamespaceKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=namespaces.vault.m.crossplane.io
type VaultNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VaultNamespaceSpec   `json:"spec"`
	Status            VaultNamespaceStatus `json:"status,omitempty"`
}

type VaultNamespaceSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              VaultNamespaceParameters `json:"forProvider"`
}

type VaultNamespaceParameters struct {
	// Name of the Vault namespace (must be unique).
	Name string `json:"name"`

	// Description of the namespace.
	Description string `json:"description,omitempty"`
}

type VaultNamespaceStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             VaultNamespaceObservation `json:"atProvider,omitempty"`
}

type VaultNamespaceObservation struct {
	Name string `json:"name,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type VaultNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultNamespace `json:"items"`
}
