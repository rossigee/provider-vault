package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const QuotaKind = "Quota"

var QuotaGroupVersionKind = SchemeGroupVersion.WithKind(QuotaKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=quota.vault.m.crossplane.io
type Quota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              QuotaSpec   `json:"spec"`
	Status            QuotaStatus `json:"status,omitempty"`
}

type QuotaSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              QuotaParameters `json:"forProvider"`
}

type QuotaParameters struct {
	// Name of the quota (must be unique within the type).
	Name string `json:"name"`

	// Type of quota: "rate" or "lease".
	Type string `json:"type"`

	// Path to apply the quota to (e.g. "sys/health", "auth/approle/login", "secret/data/myapp").
	// Use "" for global quotas.
	Path string `json:"path,omitempty"`

	// Rate is the maximum number of requests per second for rate quotas.
	// Required for rate quotas, ignored for lease quotas.
	// Use decimal notation for fractional rates (e.g. "1.5").
	Rate string `json:"rate,omitempty"`

	// MaxLeases is the maximum number of leases allowed for lease quotas.
	// Required for lease quotas, ignored for rate quotas.
	MaxLeases *int `json:"maxLeases,omitempty"`

	// Interval is the interval for rate quotas: "second", "minute", or "hour".
	// Defaults to "second".
	Interval string `json:"interval,omitempty"`

	// Blocked is a list of paths to block when the quota is exceeded.
	Blocked []string `json:"blocked,omitempty"`
}

type QuotaStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             QuotaObservation `json:"atProvider,omitempty"`
}

type QuotaObservation struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type QuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Quota `json:"items"`
}
