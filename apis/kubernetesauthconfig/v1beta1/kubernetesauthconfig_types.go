package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const KubernetesAuthConfigKind = "KubernetesAuthConfig"

var KubernetesAuthConfigGroupVersionKind = SchemeGroupVersion.WithKind(KubernetesAuthConfigKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=kubernetesauthconfig.vault.m.crossplane.io
type KubernetesAuthConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KubernetesAuthConfigSpec   `json:"spec"`
	Status            KubernetesAuthConfigStatus `json:"status,omitempty"`
}

type KubernetesAuthConfigSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              KubernetesAuthConfigParameters `json:"forProvider"`
}

type KubernetesAuthConfigParameters struct {
	// Backend is the mount path of the Kubernetes auth method (e.g. "k8s-infrastructure").
	Backend string `json:"backend"`
	// KubernetesHost is the Kubernetes API server URL.
	KubernetesHost string `json:"kubernetesHost"`
	// KubernetesCACert is the CA certificate for the Kubernetes API.
	KubernetesCACert string `json:"kubernetesCaCert,omitempty"`
	// TokenReviewerJWT is a JWT token for the token reviewer service account.
	TokenReviewerJWT string `json:"tokenReviewerJwt,omitempty"`
	// TokenReviewerJWTSecretRef references a Kubernetes secret containing the
	// JWT token for the token reviewer service account. Takes precedence over
	// TokenReviewerJWT when both are set.
	TokenReviewerJWTSecretRef *xpv1.SecretKeySelector `json:"tokenReviewerJwtSecretRef,omitempty"`
	// Issuer is the Kubernetes token issuer (defaults to kubernetes/serviceaccount).
	Issuer string `json:"issuer,omitempty"`
	// DisableISSValidation disables issuer validation.
	DisableISSValidation *bool `json:"disableIssValidation,omitempty"`
	// DisableLocalCAJWT disables local CA JWT validation.
	DisableLocalCAJWT *bool `json:"disableLocalCaJwt,omitempty"`
}

type KubernetesAuthConfigStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             KubernetesAuthConfigObservation `json:"atProvider,omitempty"`
}

type KubernetesAuthConfigObservation struct {
	Backend string `json:"backend,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type KubernetesAuthConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesAuthConfig `json:"items"`
}
