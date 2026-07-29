package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const (
	KVSecretKind = "KVSecret"
)

var KVSecretGroupVersionKind = SchemeGroupVersion.WithKind(KVSecretKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=kvsecret.vault.m.crossplane.io
type KVSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KVSecretSpec   `json:"spec"`
	Status            KVSecretStatus `json:"status,omitempty"`
}

type KVSecretSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              KVSecretParameters `json:"forProvider"`
}

type KVSecretParameters struct {
	Path      string            `json:"path"`
	Data      map[string]string `json:"data"`
	MountPath string            `json:"mountPath"`
}

type KVSecretStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             KVSecretObservation `json:"atProvider,omitempty"`
}

type KVSecretObservation struct {
	Data map[string]string `json:"data,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type KVSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KVSecret `json:"items"`
}
