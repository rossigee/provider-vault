# AuthBackendRole

**API Version**: `authbackendrole.vault.m.crossplane.io/v1beta1`

Create and manage JWT, AppRole, and Kubernetes auth roles for identity-to-policy mapping.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | Auth method mount path (e.g. `jwt-production`) |
| `forProvider.roleName` | string | yes | Name of the role |
| `forProvider.roleType` | string | no | Role type: `jwt`, `approle`, or `kubernetes` |
| `forProvider.boundAudiences` | []string | no | List of bound audiences for JWT roles |
| `forProvider.boundSubject` | string | no | Bound subject for JWT/Kubernetes roles |
| `forProvider.userClaim` | string | no | User claim for JWT roles |
| `forProvider.groupsClaim` | string | no | Groups claim for JWT roles |
| `forProvider.policies` | []string | no | Legacy policies to attach |
| `forProvider.tokenPolicies` | []string | no | Token policies to attach |
| `forProvider.tokenTtl` | int (seconds) | no | Token TTL |
| `forProvider.tokenMaxTtl` | int (seconds) | no | Token maximum TTL |
| `forProvider.tokenPeriod` | int (seconds) | no | Token renewal period |
| `forProvider.tokenNumUses` | int | no | Maximum token uses |
| `forProvider.tokenType` | string | no | Token type (`service`, `batch`, `default`) |
| `forProvider.secretIdTtl` | int (seconds) | no | SecretID TTL (AppRole only) |
| `forProvider.secretIdNumUses` | int | no | SecretID max uses (AppRole only) |
| `forProvider.tokenBoundCidrs` | []string | no | CIDR restrictions for tokens |
| `forProvider.allowedRedirectUris` | []string | no | Allowed redirect URIs (OIDC only) |
| `forProvider.clockSkewLeeway` | int (seconds) | no | Clock skew leeway (JWT only) |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example (JWT)

```yaml
apiVersion: authbackendrole.vault.m.crossplane.io/v1beta1
kind: AuthBackendRole
metadata:
  name: jwt-flux-system
  namespace: vault
spec:
  forProvider:
    backend: jwt-production
    roleName: flux-system
    roleType: jwt
    boundAudiences:
      - https://k8s-api.cluster.example.com
    boundSubject: "system:serviceaccount:flux-system:external-secrets"
    userClaim: sub
    tokenPolicies:
      - default
      - external-secrets-production
    tokenTtl: 3600
    tokenMaxTtl: 3600
  providerConfigRef:
    name: default
```

## Example (AppRole)

```yaml
apiVersion: authbackendrole.vault.m.crossplane.io/v1beta1
kind: AuthBackendRole
metadata:
  name: approle-myapp
  namespace: vault
spec:
  forProvider:
    backend: approle
    roleName: myapp
    roleType: approle
    tokenPolicies:
      - myapp-reader
    tokenTtl: 3600
    secretIdTtl: 86400
    secretIdNumUses: 1
  providerConfigRef:
    name: default
```
