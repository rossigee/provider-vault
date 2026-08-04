package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const AzureAuthConfigKind = "AzureAuthConfig"

var AzureAuthConfigGroupVersionKind = SchemeGroupVersion.WithKind(AzureAuthConfigKind)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,vault}
// +genclient
// +genclient:namespaced
// +groupName=azureauthconfig.vault.m.crossplane.io
type AzureAuthConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AzureAuthConfigSpec   `json:"spec"`
	Status            AzureAuthConfigStatus `json:"status,omitempty"`
}

type AzureAuthConfigSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              AzureAuthConfigParameters `json:"forProvider"`
}

type AzureAuthConfigParameters struct {
	// Backend is the mount path of the Azure auth method (e.g. "azure-auth").
	Backend string `json:"backend"`

	// TenantID is the Azure tenant ID for the application registration.
	TenantID string `json:"tenantId"`

	// Resource is the resource URL for Azure AD (default: "https://management.azure.com/").
	Resource string `json:"resource,omitempty"`

	// ClientID is the application/client ID registered with Azure AD.
	ClientID string `json:"clientId,omitempty"`

	// ClientSecret is the client secret for the Azure AD application.
	// This is only needed for some authentication flows.
	ClientSecret string `json:"clientSecret,omitempty"`

	// Environment is the Azure cloud environment.
	// Valid values: "AzurePublicCloud" (default), "AzureUSGovernment", "AzureChinaCloud".
	Environment string `json:"environment,omitempty"`

	// LoginMaxUserRetry is the maximum number of retries for user login (default: 5).
	LoginMaxUserRetry *int `json:"loginMaxUserRetry,omitempty"`

	// LoginScopes are the scopes to request for login (e.g. "https://management.azure.com/.default").
	LoginScopes []string `json:"loginScopes,omitempty"`
}

type AzureAuthConfigStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             AzureAuthConfigObservation `json:"atProvider,omitempty"`
}

type AzureAuthConfigObservation struct {
	Backend string `json:"backend,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AzureAuthConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureAuthConfig `json:"items"`
}
