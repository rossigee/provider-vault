# AuthMethod

**API Version**: `authmethod.vault.m.crossplane.io/v1beta1`

Enable, configure, tune, and disable auth methods (JWT, Kubernetes, AppRole, LDAP, etc.).

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.mountPath` | string | yes | Mount path for the auth method (e.g. `github-actions`) |
| `forProvider.type` | string | yes | Auth method type (e.g. `jwt`, `kubernetes`, `approle`, `ldap`, `userpass`, `cert`) |
| `forProvider.config` | map[string]string | no | Configuration options (varies by type) |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example

```yaml
apiVersion: authmethod.vault.m.crossplane.io/v1beta1
kind: AuthMethod
metadata:
  name: github-actions
  namespace: production
spec:
  forProvider:
    mountPath: github-actions
    type: jwt
    config:
      oidc_discovery_url: https://token.actions.githubusercontent.com
      bound_issuer: https://token.actions.githubusercontent.com
  providerConfigRef:
    name: default
```

## Common Config Options

| Type | Config Key | Description |
|------|-----------|-------------|
| JWT | `oidc_discovery_url` | OIDC discovery URL |
| JWT | `bound_issuer` | Expected issuer claim |
| JWT | `jwt_validation_pubkeys` | Public keys for JWT validation |
| Kubernetes | `kubernetes_host` | Kubernetes API server URL |
| Kubernetes | `kubernetes_ca_cert` | CA certificate for Kubernetes API |
| Kubernetes | `token_reviewer_jwt` | JWT for token review |

## Tuning

When tuning an existing auth method, the `config` field is sent to the Vault API as-is. Configuration keys and values are passed directly to the `POST /v1/auth/<mountPath>/tune` endpoint.
