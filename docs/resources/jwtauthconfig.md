# JWTAuthConfig

**API Version**: `jwtauthconfig.vault.m.crossplane.io/v1beta1`

Configure JWT/OIDC authentication method for Vault.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | Auth mount path (e.g., `jwt`) |
| `forProvider.oidcDiscoveryUrl` | string | no | OIDC discovery URL |
| `forProvider.oidcClientId` | string | no | OIDC client ID |
| `forProvider.oidcClientSecret` | string | no | OIDC client secret (inline; avoid in Git) |
| `forProvider.oidcClientSecretSecretRef` | SecretKeySelector | no | Ref to K8s secret holding the OIDC client secret (preferred; takes precedence) |
| `forProvider.jwtValidationPubkeys` | string | no | PEM-encoded public keys for JWT validation |
| `forProvider.boundIssuer` | string | no | Expected issuer claim |
| `forProvider.defaultRole` | string | no | Default role for JWT logins |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example (OIDC)

```yaml
apiVersion: jwtauthconfig.vault.m.crossplane.io/v1beta1
kind: JWTAuthConfig
metadata:
  name: oidc
  namespace: vault
spec:
  forProvider:
    backend: oidc
    oidcDiscoveryUrl: https://accounts.google.com
    oidcClientId: vault-oidc-client
    oidcClientSecretSecretRef:
      name: vault-oidc-client
      namespace: vault
      key: clientSecret
    defaultRole: default
  providerConfigRef:
    name: default
```

## Example (JWT with Static Keys)

```yaml
apiVersion: jwtauthconfig.vault.m.crossplane.io/v1beta1
kind: JWTAuthConfig
metadata:
  name: jwt
  namespace: vault
spec:
  forProvider:
    backend: jwt
    jwtValidationPubkeys: |
      -----BEGIN PUBLIC KEY-----
      MC0CAQACBQD...
      -----END PUBLIC KEY-----
    boundIssuer: https://auth.example.com
    defaultRole: service
  providerConfigRef:
    name: default
```