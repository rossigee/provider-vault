package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const TokenKind = "Token"

var TokenGroupVersionKind = SchemeGroupVersion.WithKind(TokenKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=token.vault.m.crossplane.io
type Token struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TokenSpec   `json:"spec"`
	Status            TokenStatus `json:"status,omitempty"`
}

type TokenSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              TokenParameters `json:"forProvider"`
}

type TokenParameters struct {
	// RoleName is the token role to create the token from.
	RoleName string `json:"roleName,omitempty"`
	// Policies to attach to the token.
	Policies []string `json:"policies,omitempty"`
	// TTL for the token (e.g. "24h").
	TTL string `json:"ttl,omitempty"`
	// RenewBefore percentage of TTL before renewal. Defaults to 0.5 (50%).
	RenewBefore *float64 `json:"renewBefore,omitempty"`
	// NoParent creates an orphan token with no parent.
	NoParent *bool `json:"noParent,omitempty"`
	// DisplayName for the token.
	DisplayName string `json:"displayName,omitempty"`
	// Period for periodic tokens.
	Period string `json:"period,omitempty"`
	// NumUses limits the number of times the token can be used.
	NumUses *int `json:"numUses,omitempty"`
}

type TokenStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             TokenObservation `json:"atProvider,omitempty"`
}

type TokenObservation struct {
	Accessor       string   `json:"accessor,omitempty"`
	ClientToken    string   `json:"clientToken,omitempty"`
	Expiration     int64    `json:"expiration,omitempty"`
	EffectivePolicies []string `json:"effectivePolicies,omitempty"`
	Orphan         bool     `json:"orphan,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type TokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Token `json:"items"`
}
