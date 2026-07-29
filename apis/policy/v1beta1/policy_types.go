package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const PolicyKind = "Policy"

var PolicyGroupVersionKind = SchemeGroupVersion.WithKind(PolicyKind)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +genclient
// +genclient:namespaced
// +groupName=policy.vault.m.crossplane.io
type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PolicySpec   `json:"spec"`
	Status            PolicyStatus `json:"status,omitempty"`
}

type PolicySpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              PolicyParameters `json:"forProvider"`
}

type PolicyParameters struct {
	Name   string `json:"name"`
	Policy string `json:"policy"`
}

type PolicyStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             PolicyObservation `json:"atProvider,omitempty"`
}

type PolicyObservation struct {
	Name string `json:"name,omitempty"`
}

// +kubebuilder:object:root=true
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Policy `json:"items"`
}
