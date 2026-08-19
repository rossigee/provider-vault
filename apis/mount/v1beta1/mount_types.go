package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const MountKind = "Mount"

var MountGroupVersionKind = SchemeGroupVersion.WithKind(MountKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=mount.vault.m.crossplane.io
type Mount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MountSpec   `json:"spec"`
	Status            MountStatus `json:"status,omitempty"`
}

type MountSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              MountParameters `json:"forProvider"`
}

type MountParameters struct {
	Path            string            `json:"path"`
	Type            string            `json:"type"`
	Description     string            `json:"description,omitempty"`
	DefaultLeaseTTL int               `json:"defaultLeaseTtl,omitempty"`
	MaxLeaseTTL     int               `json:"maxLeaseTtl,omitempty"`
	Options         map[string]string `json:"options,omitempty"`
	Config          map[string]string `json:"config,omitempty"`
}

type MountStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             MountObservation `json:"atProvider,omitempty"`
}

type MountObservation struct {
	Path string `json:"path,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type MountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mount `json:"items"`
}
