# IdentityEntity

**API Version**: `identityentity.vault.m.crossplane.io/v1beta1`

Manage Vault identity entities with attached policies and metadata.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Name of the entity |
| `forProvider.policies` | []string | no | Policies attached to the entity |
| `forProvider.metadata` | map[string]string | no | Metadata key-value pairs |
| `forProvider.disabled` | bool | no | Disable the entity |
| `forProvider.groupIds` | []string | no | Group IDs for entity membership |

## Example

```yaml
apiVersion: identityentity.vault.m.crossplane.io/v1beta1
kind: IdentityEntity
metadata:
  name: ci-bot
  namespace: vault
spec:
  forProvider:
    name: ci-bot
    policies:
      - ci-reader
      - metrics-reader
    metadata:
      created-by: crossplane
      team: platform
  providerConfigRef:
    name: default
```
