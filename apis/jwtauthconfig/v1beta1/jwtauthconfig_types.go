package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const JWTAuthConfigKind = "JWTAuthConfig"

var JWTAuthConfigGroupVersionKind = SchemeGroupVersion.WithKind(JWTAuthConfigKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=jwtauthconfig.vault.m.crossplane.io
type JWTAuthConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JWTAuthConfigSpec   `json:"spec"`
	Status            JWTAuthConfigStatus `json:"status,omitempty"`
}

type JWTAuthConfigSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              JWTAuthConfigParameters `json:"forProvider"`
}

type JWTAuthConfigParameters struct {
	// Backend is the mount path of the JWT/OIDC auth method (e.g. "jwt-auth").
	Backend string `json:"backend"`

	// --- OIDC configuration ---

	// OIDCDiscoveryURL is the OIDC discovery URL for the provider (e.g. "https://accounts.google.com").
	OIDCDiscoveryURL string `json:"oidcDiscoveryUrl,omitempty"`
	// OIDCDiscoveryCAPEM is the PEM-encoded CA certificate for the OIDC discovery URL.
	OIDCDiscoveryCAPEM string `json:"oidcDiscoveryCaPem,omitempty"`
	// OIDCClientID is the OAuth2 client ID for OIDC authentication.
	OIDCClientID string `json:"oidcClientId,omitempty"`
	// OIDCClientSecret is the OAuth2 client secret for OIDC authentication.
	OIDCClientSecret string `json:"oidcClientSecret,omitempty"`

	// --- JWT configuration ---

	// JWTValidationPubKeys is a list of PEM-encoded public keys used to validate JWTs.
	JWTValidationPubKeys []string `json:"jwtValidationPubKeys,omitempty"`
	// JWKSCacheDuration is the duration to cache JWKS responses (e.g. "1h"). Zero disables caching.
	JWKSCacheDuration string `json:"jwksCacheDuration,omitempty"`

	// --- Common configuration ---

	// BoundIssuer is the issuer (iss claim) that the provider uses to validate incoming tokens.
	BoundIssuer string `json:"boundIssuer,omitempty"`
	// DefaultRole is the name of the default role to use if no role is specified in the request.
	DefaultRole string `json:"defaultRole,omitempty"`

	// --- TTL and lease configuration ---

	// MaxTTL is the maximum duration a token issued by this method is valid (e.g. "24h"). Zero means no max.
	MaxTTL string `json:"maxTtl,omitempty"`
	// TTL is the default duration a token issued by this method is valid (e.g. "1h").
	TTL string `json:"ttl,omitempty"`
	// Period is the period a token issued by this method is valid.
	Period string `json:"period,omitempty"`
}

type JWTAuthConfigStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             JWTAuthConfigObservation `json:"atProvider,omitempty"`
}

type JWTAuthConfigObservation struct {
	Backend string `json:"backend,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type JWTAuthConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JWTAuthConfig `json:"items"`
}
