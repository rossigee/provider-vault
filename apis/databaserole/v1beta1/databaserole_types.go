package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const DatabaseRoleKind = "DatabaseRole"

var DatabaseRoleGroupVersionKind = SchemeGroupVersion.WithKind(DatabaseRoleKind)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=databaserole.vault.m.crossplane.io
type DatabaseRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DatabaseRoleSpec   `json:"spec"`
	Status            DatabaseRoleStatus `json:"status,omitempty"`
}

type DatabaseRoleSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              DatabaseRoleParameters `json:"forProvider"`
}

type DatabaseRoleParameters struct {
	Backend          string   `json:"backend"`
	Name             string   `json:"name"`
	DBName           string   `json:"dbName"`
	CreationStatements []string `json:"creationStatements,omitempty"`
	RevocationStatements []string `json:"revocationStatements,omitempty"`
	RollbackStatements  []string `json:"rollbackStatements,omitempty"`
	RenewStatements     []string `json:"renewStatements,omitempty"`
	DefaultTTL         string   `json:"defaultTtl,omitempty"`
	MaxTTL             string   `json:"maxTtl,omitempty"`
}

type DatabaseRoleStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             DatabaseRoleObservation `json:"atProvider,omitempty"`
}

type DatabaseRoleObservation struct {
	Name string `json:"name,omitempty"`
}

// +kubebuilder:object:root=true
type DatabaseRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseRole `json:"items"`
}
