package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const TransitKeyKind = "TransitKey"

var TransitKeyGroupVersionKind = SchemeGroupVersion.WithKind(TransitKeyKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=transitkey.vault.m.crossplane.io
type TransitKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TransitKeySpec   `json:"spec"`
	Status            TransitKeyStatus `json:"status,omitempty"`
}

type TransitKeySpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              TransitKeyParameters `json:"forProvider"`
}

type TransitKeyParameters struct {
	Backend               string   `json:"backend"`
	Name                  string   `json:"name"`
	Type                  string   `json:"type,omitempty"`
	ConvergentEncryption  *bool    `json:"convergentEncryption,omitempty"`
	Derived               *bool    `json:"derived,omitempty"`
	Exportable            *bool    `json:"exportable,omitempty"`
	AllowPlaintextBackup  *bool    `json:"allowPlaintextBackup,omitempty"`
	AutoRotatePeriod      string   `json:"autoRotatePeriod,omitempty"`
	MinDecryptionVersion  *int     `json:"minDecryptionVersion,omitempty"`
	MinEncryptionVersion  *int     `json:"minEncryptionVersion,omitempty"`
}

type TransitKeyStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             TransitKeyObservation `json:"atProvider,omitempty"`
}

type TransitKeyObservation struct {
	Name              string `json:"name,omitempty"`
	Type              string `json:"type,omitempty"`
	DeletionAllowed   bool   `json:"deletionAllowed,omitempty"`
	MinDecryptionVersion int `json:"minDecryptionVersion,omitempty"`
	LatestVersion     int    `json:"latestVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type TransitKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TransitKey `json:"items"`
}
