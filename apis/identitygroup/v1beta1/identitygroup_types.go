package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const IdentityGroupKind = "IdentityGroup"

var IdentityGroupGroupVersionKind = SchemeGroupVersion.WithKind(IdentityGroupKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=identitygroup.vault.m.crossplane.io
type IdentityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              IdentityGroupSpec   `json:"spec"`
	Status            IdentityGroupStatus `json:"status,omitempty"`
}

type IdentityGroupSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              IdentityGroupParameters `json:"forProvider"`
}

type IdentityGroupParameters struct {
	// Name of the group.
	Name string `json:"name"`
	// Type of group: "internal" or "external".
	Type string `json:"type,omitempty"`
	// Policies attached to the group.
	Policies []string `json:"policies,omitempty"`
	// MemberEntityIDs list of entity IDs that are members of the group.
	MemberEntityIDs []string `json:"memberEntityIds,omitempty"`
	// Metadata for the group.
	Metadata map[string]string `json:"metadata,omitempty"`
}

type IdentityGroupStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             IdentityGroupObservation `json:"atProvider,omitempty"`
}

type IdentityGroupObservation struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type IdentityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IdentityGroup `json:"items"`
}
