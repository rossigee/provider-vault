package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const IdentityEntityKind = "IdentityEntity"

var IdentityEntityGroupVersionKind = SchemeGroupVersion.WithKind(IdentityEntityKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=identityentity.vault.m.crossplane.io
type IdentityEntity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              IdentityEntitySpec   `json:"spec"`
	Status            IdentityEntityStatus `json:"status,omitempty"`
}

type IdentityEntitySpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              IdentityEntityParameters `json:"forProvider"`
}

type IdentityEntityParameters struct {
	// Name of the entity.
	Name string `json:"name"`
	// Policies attached to the entity.
	Policies []string `json:"policies,omitempty"`
	// Metadata for the entity.
	Metadata map[string]string `json:"metadata,omitempty"`
	// Disabled disables the entity.
	Disabled *bool `json:"disabled,omitempty"`
	// GroupIDs for entity membership.
	GroupIDs []string `json:"groupIds,omitempty"`
}

type IdentityEntityStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             IdentityEntityObservation `json:"atProvider,omitempty"`
}

type IdentityEntityObservation struct {
	Name      string `json:"name,omitempty"`
	ID        string `json:"id,omitempty"`
	CanonicalID string `json:"canonicalId,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type IdentityEntityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IdentityEntity `json:"items"`
}
