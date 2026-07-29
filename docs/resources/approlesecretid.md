# AppRoleSecretID

**API Version**: `approlesecretid.vault.m.crossplane.io/v1beta1`

Generate an AppRole SecretID for machine-to-machine authentication. The generated SecretID and accessor are automatically written to a Kubernetes connection secret.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | AppRole auth mount path (e.g. `approle`) |
| `forProvider.roleName` | string | yes | AppRole role name to generate the SecretID for |
| `forProvider.metadata` | map[string]string | no | Key-value metadata (Vault stores as JSON string) |
| `forProvider.cidrList` | []string | no | CIDR blocks that can use this SecretID |
| `forProvider.tokenBoundCidrs` | []string | no | CIDR restrictions for tokens created with this SecretID |
| `writeConnectionSecretToRef.name` | string | yes | Name of the secret to write credentials to |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Connection Secret

The generated Secret contains:

| Key | Description |
|-----|-------------|
| `secret_id` | The Vault SecretID (used with RoleID for AppRole authentication) |
| `secret_id_accessor` | The SecretID accessor (for management operations like lookup and destroy) |

## Example

```yaml
apiVersion: approlesecretid.vault.m.crossplane.io/v1beta1
kind: AppRoleSecretID
metadata:
  name: my-app-secretid
  namespace: production
spec:
  forProvider:
    backend: approle
    roleName: my-app
    metadata:
      created-by: crossplane
      environment: production
    cidrList:
      - "10.0.0.0/8"
  writeConnectionSecretToRef:
    name: my-app-secretid
  providerConfigRef:
    name: default
```

## Usage with Consuming Applications

Applications authenticate to Vault using AppRole by combining:
- **RoleID**: Retrieved from the AppRole configuration (out of scope for this resource)
- **SecretID**: Retrieved from the connection secret referenced by `writeConnectionSecretToRef`

## Deletion

On deletion, the provider destroys the SecretID in Vault before removing the Kubernetes resource:

1. **Primary path**: Uses the `secret-id-accessor/destroy` endpoint with the accessor from the `crossplane.io/external-name` annotation (always available, does not depend on the connection secret)
2. **Fallback**: Recovers the raw `secret_id` from `status.atProvider`, the connection secret, or via Vault lookup, then calls `secret-id/destroy`

The accessor-based destroy (primary path) ensures reliable cleanup even when the connection secret has been deleted before the managed resource.

## Lifecycle

1. **Create**: Generates a new SecretID in Vault, stores the accessor in the `crossplane.io/external-name` annotation, writes `secret_id` and `secret_id_accessor` to the connection secret
2. **Observe**: Looks up the SecretID by accessor using the `secret-id-accessor/lookup` endpoint
3. **Update**: Destroys the old SecretID (by accessor), generates a new one, updates the connection secret
4. **Delete**: Destroys the SecretID by accessor (primary) or by secret_id (fallback)
