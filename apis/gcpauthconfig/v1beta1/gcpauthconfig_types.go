package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const GCPAuthConfigKind = "GCPAuthConfig"

var GCPAuthConfigGroupVersionKind = SchemeGroupVersion.WithKind(GCPAuthConfigKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=gcpauthconfig.vault.m.crossplane.io
type GCPAuthConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GCPAuthConfigSpec   `json:"spec"`
	Status            GCPAuthConfigStatus `json:"status,omitempty"`
}

type GCPAuthConfigSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              GCPAuthConfigParameters `json:"forProvider"`
}

type GCPAuthConfigParameters struct {
	// Backend is the mount path of the GCP auth method (e.g. "gcp-auth").
	Backend string `json:"backend"`

	// Credentials is a JSON-encoded GCP service account key for API calls.
	// Required if not using GCE metadata-based authentication.
	Credentials string `json:"credentials,omitempty"`

	// ServiceAccountEmail is the email of the GCP service account used to
	// impersonate during authentication. If set, Vault impersonates this account.
	ServiceAccountEmail string `json:"serviceAccountEmail,omitempty"`

	// ProjectID is the GCP project ID. Used for IAM role-based auth.
	ProjectID string `json:"projectId,omitempty"`

	// Zone is the GCP zone. Used for GCE-based authentication.
	Zone string `json:"zone,omitempty"`

	// ClusterName is the GKE cluster name. Used for GCE-based authentication.
	ClusterName string `json:"clusterName,omitempty"`

	// MaxTTL is the maximum duration a token issued by this method is valid (e.g. "24h").
	MaxTTL string `json:"maxTtl,omitempty"`
}

type GCPAuthConfigStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             GCPAuthConfigObservation `json:"atProvider,omitempty"`
}

type GCPAuthConfigObservation struct {
	Backend string `json:"backend,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type GCPAuthConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GCPAuthConfig `json:"items"`
}
