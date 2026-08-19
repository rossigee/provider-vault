package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const LDAPAuthConfigKind = "LDAPAuthConfig"

var LDAPAuthConfigGroupVersionKind = SchemeGroupVersion.WithKind(LDAPAuthConfigKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=ldapauthconfig.vault.m.crossplane.io
type LDAPAuthConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              LDAPAuthConfigSpec   `json:"spec"`
	Status            LDAPAuthConfigStatus `json:"status,omitempty"`
}

type LDAPAuthConfigSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              LDAPAuthConfigParameters `json:"forProvider"`
}

type LDAPAuthConfigParameters struct {
	// Backend is the mount path of the LDAP auth method (e.g. "ldap-auth").
	Backend string `json:"backend"`

	// URL of the LDAP server (e.g. "ldap://ldap.example.com:389" or "ldaps://ldap.example.com:636").
	URL string `json:"url"`

	// BindDN is the DN of the account used to perform searches (e.g. "cn=vault,ou=apps,dc=example,dc=com").
	BindDN string `json:"bindDn"`

	// BindPass is the password for the bind DN.
	BindPass string `json:"bindPass"`

	// Certificate is the PEM-encoded CA certificate for the LDAP server.
	Certificate string `json:"certificate,omitempty"`

	// RequestTimeout is the timeout in seconds for LDAP requests (default 30).
	RequestTimeout *int `json:"requestTimeout,omitempty"`

	// StartTLS enables StartTLS on the LDAP connection.
	StartTLS *bool `json:"startTls,omitempty"`

	// InsecureTLS skips server certificate verification (not recommended for production).
	InsecureTLS *bool `json:"insecureTls,omitempty"`

	// UserDN is the base DN to start user search from (e.g. "ou=users,dc=example,dc=com").
	UserDN string `json:"userDn,omitempty"`

	// UserAttr is the attribute to match the username against (default "cn").
	UserAttr string `json:"userAttr,omitempty"`

	// GroupDN is the base DN to start group search from (e.g. "ou=groups,dc=example,dc=com").
	GroupDN string `json:"groupDn,omitempty"`

	// GroupAttr is the attribute to match groups against (default "member").
	GroupAttr string `json:"groupAttr,omitempty"`

	// UPNDomain is the domain to use for UPN-style login (e.g. "example.com").
	UPNDomain string `json:"upnDomain,omitempty"`

	// DiscoverDN discovers the user's DN from user-attr (useful with Active Directory).
	DiscoverDN *bool `json:"discoverDn,omitempty"`

	// UserFilter is a custom user search filter (e.g. "(&({{.UserAttr}}={{.Username}})(!(objectClass=accountEnabled=FALSE)))").
	UserFilter string `json:"userFilter,omitempty"`

	// GroupFilter is a custom group search filter.
	GroupFilter string `json:"groupFilter,omitempty"`

	// MaxTTL is the maximum duration a token issued by this method is valid (e.g. "24h").
	MaxTTL string `json:"maxTtl,omitempty"`

	// TLSMinVersion is the minimum TLS version for the LDAP connection (e.g. "tls12").
	TLSMinVersion string `json:"tlsMinVersion,omitempty"`

	// TLSMaxVersion is the maximum TLS version for the LDAP connection (e.g. "tls13").
	TLSMaxVersion string `json:"tlsMaxVersion,omitempty"`
}

type LDAPAuthConfigStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             LDAPAuthConfigObservation `json:"atProvider,omitempty"`
}

type LDAPAuthConfigObservation struct {
	Backend string `json:"backend,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type LDAPAuthConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPAuthConfig `json:"items"`
}
