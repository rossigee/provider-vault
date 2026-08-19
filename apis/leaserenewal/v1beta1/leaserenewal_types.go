package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const LeaseRenewalKind = "LeaseRenewal"

var LeaseRenewalGroupVersionKind = SchemeGroupVersion.WithKind(LeaseRenewalKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=leaserenewal.vault.m.crossplane.io
type LeaseRenewal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              LeaseRenewalSpec   `json:"spec"`
	Status            LeaseRenewalStatus `json:"status,omitempty"`
}

type LeaseRenewalSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              LeaseRenewalParameters `json:"forProvider"`
}

type LeaseRenewalParameters struct {
	// LeaseID is the Vault lease identifier to renew.
	// This is obtained when reading a secret from Vault.
	LeaseID string `json:"leaseID"`

	// Increment is the requested amount of time (in seconds) to extend the lease.
	// If not specified, the default increment is used (typically the original lease duration).
	// +optional
	Increment *int `json:"increment,omitempty"`

	// RevokeOnDelete specifies whether to revoke the lease when this resource is deleted.
	// Defaults to false.
	// +optional
	RevokeOnDelete *bool `json:"revokeOnDelete,omitempty"`
}

type LeaseRenewalStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             LeaseRenewalObservation `json:"atProvider,omitempty"`
}

type LeaseRenewalObservation struct {
	// LeaseID is the current lease identifier.
	LeaseID string `json:"leaseID,omitempty"`

	// Renewable indicates whether the lease is renewable.
	Renewable bool `json:"renewable,omitempty"`

	// TTL is the remaining time-to-live of the lease in seconds.
	TTL int `json:"ttl,omitempty"`

	// LastRenewed is the timestamp of the last successful renewal.
	// +optional
	LastRenewed *metav1.Time `json:"lastRenewed,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type LeaseRenewalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LeaseRenewal `json:"items"`
}
