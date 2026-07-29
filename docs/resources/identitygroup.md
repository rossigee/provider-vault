# IdentityGroup

**API Version**: `identitygroup.vault.m.crossplane.io/v1beta1`

Manage Vault identity groups with member entities and attached policies.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Name of the group |
| `forProvider.type` | string | no | Group type: `internal` (default) or `external` |
| `forProvider.policies` | []string | no | Policies attached to the group |
| `forProvider.memberEntityIds` | []string | no | Entity IDs that are members of the group |
| `forProvider.metadata` | map[string]string | no | Metadata key-value pairs |

## Example

```yaml
apiVersion: identitygroup.vault.m.crossplane.io/v1beta1
kind: IdentityGroup
metadata:
  name: platform-engineers
  namespace: vault
spec:
  forProvider:
    name: platform-engineers
    type: internal
    policies:
      - vault-admin
      - audit-reader
    memberEntityIds:
      - "entity-1234-..."
      - "entity-5678-..."
    metadata:
      team: platform
      slack-channel: "#vault-admins"
  providerConfigRef:
    name: default
```
