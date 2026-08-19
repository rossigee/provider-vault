package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AWSAuthConfigKind = "AWSAuthConfig"

var AWSAuthConfigGroupVersionKind = SchemeGroupVersion.WithKind(AWSAuthConfigKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=awsauthconfig.vault.m.crossplane.io
type AWSAuthConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AWSAuthConfigSpec   `json:"spec"`
	Status            AWSAuthConfigStatus `json:"status,omitempty"`
}

type AWSAuthConfigSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              AWSAuthConfigParameters `json:"forProvider"`
}

type AWSAuthConfigParameters struct {
	// Backend is the mount path of the AWS auth method (e.g. "aws-auth").
	Backend string `json:"backend"`

	// IAMServerIDHeaderValue is the value to require in the X-Vault-AWS-IAM-Server-ID header.
	// This provides defense against replay attacks.
	IAMServerIDHeaderValue string `json:"iamServerIdHeaderValue,omitempty"`

	// IAMRequestURL is the base URL to use for the iam_server_id_header_value request.
	// This is needed for some cloud providers where the AWS STS endpoint is not available.
	IAMRequestURL string `json:"iamRequestUrl,omitempty"`

	// IAMRequestPayload is the payload to use for the IAM auth request.
	// Only used if the IAM auth method requires a custom payload format.
	IAMRequestPayload string `json:"iamRequestPayload,omitempty"`

	// STSEndpoint is a custom endpoint URL for the AWS STS service.
	// This is useful in environments where the default STS endpoint is not available.
	STSEndpoint string `json:"stsEndpoint,omitempty"`

	// STSFallbackRegion is the AWS region to use if the request's region cannot be determined.
	STSFallbackRegion string `json:"stsFallbackRegion,omitempty"`

	// STSDisableRedirect disables the redirect behavior when calling STS.
	STSDisableRedirect *bool `json:"stsDisableRedirect,omitempty"`

	// MaxTTL is the maximum duration a token issued by this method is valid (e.g. "24h").
	MaxTTL string `json:"maxTtl,omitempty"`

	// AccessIdentity is the identifier for the identity used to sign requests.
	// This is useful when multiple AWS accounts need to authenticate against a single Vault cluster.
	AccessIdentity string `json:"accessIdentity,omitempty"`
}

type AWSAuthConfigStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             AWSAuthConfigObservation `json:"atProvider,omitempty"`
}

type AWSAuthConfigObservation struct {
	Backend string `json:"backend,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AWSAuthConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSAuthConfig `json:"items"`
}
