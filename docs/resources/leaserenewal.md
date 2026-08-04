# LeaseRenewal

**API Version**: `leaserenewal.vault.m.crossplane.io/v1beta1`

Automatically renew Vault leases to prevent expiration.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.leaseID` | string | yes | The Vault lease ID to renew |
| `forProvider.increment` | int | no | Seconds to extend the lease |
| `forProvider.revokeOnDelete` | bool | no | Revoke lease when resource is deleted |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example

```yaml
apiVersion: leaserenewal.vault.m.crossplane.io/v1beta1
kind: LeaseRenewal
metadata:
  name: db-creds-renewal
  namespace: vault
spec:
  forProvider:
    leaseID: database/creds/my-role/abcd1234...
    increment: 3600
    revokeOnDelete: true
  providerConfigRef:
    name: default
```

## Example (Token Renewal)

```yaml
apiVersion: leaserenewal.vault.m.crossplane.io/v1beta1
kind: LeaseRenewal
metadata:
  name: token-renewal
  namespace: vault
spec:
  forProvider:
    leaseID: auth/token/create/my-role/abcd1234...
    increment: 7200
    revokeOnDelete: true
  providerConfigRef:
    name: default
```