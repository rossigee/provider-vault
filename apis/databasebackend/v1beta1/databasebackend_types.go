package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const DatabaseBackendKind = "DatabaseBackend"

var DatabaseBackendGroupVersionKind = SchemeGroupVersion.WithKind(DatabaseBackendKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=databasebackend.vault.m.crossplane.io
type DatabaseBackend struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DatabaseBackendSpec   `json:"spec"`
	Status            DatabaseBackendStatus `json:"status,omitempty"`
}

type DatabaseBackendSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              DatabaseBackendParameters `json:"forProvider"`
}

type DatabaseBackendParameters struct {
	// Backend is the mount path of the database engine (e.g. "postgres").
	Backend string `json:"backend"`
	// Name for this database connection configuration.
	Name string `json:"name"`
	// PluginName is the database plugin to use (e.g. "postgresql-database-plugin").
	PluginName string `json:"pluginName,omitempty"`
	// ConnectionURL for the database server. Uses {{username}} and {{password}} templates.
	ConnectionURL string `json:"connectionUrl"`
	// Username for Vault to authenticate to the database.
	Username string `json:"username"`
	// Password for Vault to authenticate to the database.
	Password string `json:"password"`
	// AllowedRoles restricts which roles can use this connection.
	AllowedRoles []string `json:"allowedRoles,omitempty"`
	// MaxConnectionLifetime for database connections.
	MaxConnectionLifetime string `json:"maxConnectionLifetime,omitempty"`
	// MaxIdleConnections for the connection pool.
	MaxIdleConnections *int `json:"maxIdleConnections,omitempty"`
	// MaxOpenConnections for the connection pool.
	MaxOpenConnections *int `json:"maxOpenConnections,omitempty"`
	// VerifyConnection tests the database connection on configuration.
	VerifyConnection *bool `json:"verifyConnection,omitempty"`
}

type DatabaseBackendStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             DatabaseBackendObservation `json:"atProvider,omitempty"`
}

type DatabaseBackendObservation struct {
	Name string `json:"name,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type DatabaseBackendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseBackend `json:"items"`
}
