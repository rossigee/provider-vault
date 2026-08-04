# AzureAuthConfig

**API Version**: `azureauthconfig.vault.m.crossplane.io/v1beta1`

Configure Azure MSI authentication method for Vault.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | Auth mount path (e.g., `azure`) |
| `forProvider.tenantId` | string | yes | Azure tenant ID |
| `forProvider.clientId` | string | no | Azure client ID |
| `forProvider.clientSecret` | string | no | Azure client secret (use secretRef in production) |
| `forProvider.environment` | string | no | Azure cloud environment (default: `AzurePublic`) |
| `forProvider.resource` | string | no | Resource to request token for |
| `forProvider.scopes` | string | no | Comma-separated scopes |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example

```yaml
apiVersion: azureauthconfig.vault.m.crossplane.io/v1beta1
kind: AzureAuthConfig
metadata:
  name: azure
  namespace: vault
spec:
  forProvider:
    backend: azure
    tenantId: 12345678-1234-1234-1234-123456789012
    clientId: 12345678-1234-1234-1234-123456789012
    resource: https://management.azure.com
    environment: AzurePublic
  providerConfigRef:
    name: default
```

## Example (Azure Government)

```yaml
apiVersion: azureauthconfig.vault.m.crossplane.io/v1beta1
kind: AzureAuthConfig
metadata:
  name: azure-gov
  namespace: vault
spec:
  forProvider:
    backend: azure-gov
    tenantId: 12345678-1234-1234-1234-123456789012
    clientId: 12345678-1234-1234-1234-123456789012
    resource: https://management.usgovcloudapi.net
    environment: AzureGovernment
  providerConfigRef:
    name: default
```