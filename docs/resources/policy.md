# Policy

**API Version**: `policy.vault.m.crossplane.io/v1beta1`

Create and manage Vault ACL policies.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Policy name |
| `forProvider.policy` | string | yes | HCL policy definition |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example

```yaml
apiVersion: policy.vault.m.crossplane.io/v1beta1
kind: Policy
metadata:
  name: app-reader
  namespace: production
spec:
  forProvider:
    name: app-reader
    policy: |
      path "secret/data/app/*" {
        capabilities = ["read", "list"]
      }
      path "secret/metadata/app/*" {
        capabilities = ["list"]
      }
  providerConfigRef:
    name: default
```

## Common Patterns

### Read-only access to a KV path
```hcl
path "secret/data/app/*" {
  capabilities = ["read", "list"]
}
```

### Admin access with create and delete
```hcl
path "secret/data/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
```

### PKI certificate issuance
```hcl
path "pki-production-ecdsa/issue/*" {
  capabilities = ["create", "update"]
}
```

### Identity management
```hcl
path "identity/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
```
