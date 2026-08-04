# Namespace

**API Version**: `namespace.vault.m.crossplane.io/v1beta1`

Create and manage Vault namespaces for isolation.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Namespace name |
| `forProvider.description` | string | no | Human-readable description |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example

```yaml
apiVersion: namespace.vault.m.crossplane.io/v1beta1
kind: Namespace
metadata:
  name: production
  namespace: vault
spec:
  forProvider:
    name: production
    description: Production namespace for core services
  providerConfigRef:
    name: default
```

## Example (Nested Namespace)

```yaml
apiVersion: namespace.vault.m.crossplane.io/v1beta1
kind: Namespace
metadata:
  name: production-team-a
  namespace: vault
spec:
  forProvider:
    name: production/team-a
    description: Team A production namespace
  providerConfigRef:
    name: default
```