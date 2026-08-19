package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AuthBackendRoleKind = "AuthBackendRole"

var AuthBackendRoleGroupVersionKind = SchemeGroupVersion.WithKind(AuthBackendRoleKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=authbackendrole.vault.m.crossplane.io
type AuthBackendRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AuthBackendRoleSpec   `json:"spec"`
	Status            AuthBackendRoleStatus `json:"status,omitempty"`
}

type AuthBackendRoleSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              AuthBackendRoleParameters `json:"forProvider"`
}

type AuthBackendRoleParameters struct {
	Backend                       string   `json:"backend"`
	RoleName                      string   `json:"roleName"`
	RoleType                      string   `json:"roleType,omitempty"`
	BoundAudiences                []string `json:"boundAudiences,omitempty"`
	BoundSubject                  string   `json:"boundSubject,omitempty"`
	BoundServiceAccountNames      []string `json:"boundServiceAccountNames,omitempty"`
	BoundServiceAccountNamespaces []string `json:"boundServiceAccountNamespaces,omitempty"`
	UserClaim                     string   `json:"userClaim,omitempty"`
	GroupsClaim                   string   `json:"groupsClaim,omitempty"`
	Policies                      []string `json:"policies,omitempty"`
	TokenPolicies                 []string `json:"tokenPolicies,omitempty"`
	TokenTTL                      *int     `json:"tokenTtl,omitempty"`
	TokenMaxTTL                   *int     `json:"tokenMaxTtl,omitempty"`
	TokenPeriod                   *int     `json:"tokenPeriod,omitempty"`
	TokenNumUses                  *int     `json:"tokenNumUses,omitempty"`
	TokenType                     string   `json:"tokenType,omitempty"`
	SecretIDTTL                   *int     `json:"secretIdTtl,omitempty"`
	SecretIDNumUses               *int     `json:"secretIdNumUses,omitempty"`
	TokenBoundCIDRs               []string `json:"tokenBoundCidrs,omitempty"`
	AllowedRedirectURIs           []string `json:"allowedRedirectUris,omitempty"`
	ClockSkewLeeway               *int     `json:"clockSkewLeeway,omitempty"`
}

type AuthBackendRoleStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             AuthBackendRoleObservation `json:"atProvider,omitempty"`
}

type AuthBackendRoleObservation struct {
	RoleName string `json:"roleName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AuthBackendRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthBackendRole `json:"items"`
}
