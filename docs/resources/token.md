# Token

**API Version**: `token.vault.m.crossplane.io/v1beta1`

Create, renew, and revoke Vault tokens. Supports automatic renewal based on TTL percentage. The token is written to a Kubernetes connection secret.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.roleName` | string | no | Token role to create the token from |
| `forProvider.policies` | []string | no | Policies to attach to the token |
| `forProvider.ttl` | string | no | Token TTL (e.g. `24h`) |
| `forProvider.renewBefore` | float | no | Renew when this fraction of TTL remains (default: `0.5`, i.e. 50%) |
| `forProvider.noParent` | bool | no | Create an orphan token with no parent |
| `forProvider.displayName` | string | no | Display name for the token |
| `forProvider.period` | string | no | Period for periodic tokens |
| `forProvider.numUses` | int | no | Limit the number of times the token can be used |
| `writeConnectionSecretToRef.name` | string | yes | Name of the secret to write the token to |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Connection Secret

| Key | Description |
|-----|-------------|
| `token` | The Vault token |

## Example

```yaml
apiVersion: token.vault.m.crossplane.io/v1beta1
kind: Token
metadata:
  name: external-secrets-token
  namespace: production
spec:
  forProvider:
    roleName: external-secrets
    policies:
      - external-secrets-reader
    ttl: 24h
    renewBefore: 0.5
    displayName: external-secrets-prod
  writeConnectionSecretToRef:
    name: external-secrets-token
  providerConfigRef:
    name: default
```

## Auto-Renewal

The Token resource automatically renews when the remaining TTL drops below `renewBefore * ttl`. For example, with a 24-hour TTL and `renewBefore: 0.5`, renewal triggers at 12 hours (50% of TTL remaining).

The connection secret is updated with the new token value on each renewal.
