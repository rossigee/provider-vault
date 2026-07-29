package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const AuthMethodKind = "AuthMethod"

var AuthMethodGroupVersionKind = SchemeGroupVersion.WithKind(AuthMethodKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=authmethod.vault.m.crossplane.io
type AuthMethod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AuthMethodSpec   `json:"spec"`
	Status            AuthMethodStatus `json:"status,omitempty"`
}

type AuthMethodSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              AuthMethodParameters `json:"forProvider"`
}

type AuthMethodParameters struct {
	MountPath string            `json:"mountPath"`
	Type      string            `json:"type"`
	Config    map[string]string `json:"config,omitempty"`
}

type AuthMethodStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             AuthMethodObservation `json:"atProvider,omitempty"`
}

type AuthMethodObservation struct {
	MountPath string `json:"mountPath,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AuthMethodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthMethod `json:"items"`
}
