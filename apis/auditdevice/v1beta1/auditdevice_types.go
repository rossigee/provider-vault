package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const AuditDeviceKind = "AuditDevice"

var AuditDeviceGroupVersionKind = SchemeGroupVersion.WithKind(AuditDeviceKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=auditdevice.vault.m.crossplane.io
type AuditDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AuditDeviceSpec   `json:"spec"`
	Status            AuditDeviceStatus `json:"status,omitempty"`
}

type AuditDeviceSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              AuditDeviceParameters `json:"forProvider"`
}

type AuditDeviceParameters struct {
	// Path is the mount path of the audit device (e.g. "file-audit").
	Path string `json:"path"`

	// Type of audit device: "file", "socket", "syslog", "tcp", or "http".
	Type string `json:"type"`

	// Description of the audit device.
	Description string `json:"description,omitempty"`

	// Local indicates this is a local-only audit device, not replicated.
	Local *bool `json:"local,omitempty"`

	// Options for the audit device (e.g. file_path, address, hmac)
	Options map[string]string `json:"options,omitempty"`
}

type AuditDeviceStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             AuditDeviceObservation `json:"atProvider,omitempty"`
}

type AuditDeviceObservation struct {
	// Path is the mount path of the audit device.
	Path string `json:"path,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AuditDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuditDevice `json:"items"`
}
