package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AppRoleSecretIDKind = "AppRoleSecretID"

var AppRoleSecretIDGroupVersionKind = SchemeGroupVersion.WithKind(AppRoleSecretIDKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=approlesecretid.vault.m.crossplane.io
type AppRoleSecretID struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AppRoleSecretIDSpec   `json:"spec"`
	Status            AppRoleSecretIDStatus `json:"status,omitempty"`
}

type AppRoleSecretIDSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              AppRoleSecretIDParameters `json:"forProvider"`
}

type AppRoleSecretIDParameters struct {
	// Backend is the mount path of the AppRole auth method (e.g. "approle").
	Backend string `json:"backend"`
	// RoleName is the name of the AppRole backend role.
	RoleName string `json:"roleName"`
	// Metadata to associate with the SecretID.
	Metadata map[string]string `json:"metadata,omitempty"`
	// CidrList of CIDR blocks that can use this SecretID.
	CidrList []string `json:"cidrList,omitempty"`
	// TokenBoundCIDRs restricts token use to specific CIDRs.
	TokenBoundCIDRs []string `json:"tokenBoundCidrs,omitempty"`
}

type AppRoleSecretIDStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             AppRoleSecretIDObservation `json:"atProvider,omitempty"`
}

type AppRoleSecretIDObservation struct {
	SecretIDAccessor string `json:"secretIdAccessor,omitempty"`
	SecretID         string `json:"secretId,omitempty"`
	// RoleID is the public identifier of the AppRole role. It is static per
	// role and is combined with a SecretID to authenticate against Vault.
	RoleID string `json:"roleId,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AppRoleSecretIDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppRoleSecretID `json:"items"`
}
