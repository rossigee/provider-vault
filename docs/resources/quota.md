# Quota

**API Version**: `quota.vault.m.crossplane.io/v1beta1`

Manage rate and lease quotas in Vault to limit request rates and lease counts.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Unique quota name |
| `forProvider.type` | string | yes | Quota type: `rate` or `lease` |
| `forProvider.path` | string | no | Path to apply quota to (empty for global) |
| `forProvider.rate` | string | no | Requests per second for rate quotas (e.g., "100") |
| `forProvider.maxLeases` | int | no | Maximum number of leases for lease quotas |
| `forProvider.interval` | string | no | Rate interval: `second`, `minute`, or `hour` |
| `forProvider.blocked` | []string | no | Paths to block when quota exceeded |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example (Rate Quota)

```yaml
apiVersion: quota.vault.m.crossplane.io/v1beta1
kind: Quota
metadata:
  name: global-rate-limit
  namespace: vault
spec:
  forProvider:
    name: global-rate-limit
    type: rate
    path: ""
    rate: "100"
    interval: second
  providerConfigRef:
    name: default
```

## Example (Lease Quota)

```yaml
apiVersion: quota.vault.m.crossplane.io/v1beta1
kind: Quota
metadata:
  name: database-leases
  namespace: vault
spec:
  forProvider:
    name: database-leases
    type: lease
    path: database/
    maxLeases: 1000
  providerConfigRef:
    name: default
```