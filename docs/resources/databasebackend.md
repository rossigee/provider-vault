# DatabaseBackend

**API Version**: `databasebackend.vault.m.crossplane.io/v1beta1`

Configure database connections for Vault's database secrets engine (PostgreSQL, MySQL, MongoDB, etc.).

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | Mount path of the database engine |
| `forProvider.name` | string | yes | Name for this database connection configuration |
| `forProvider.pluginName` | string | no | Database plugin (e.g. `postgresql-database-plugin`) |
| `forProvider.connectionUrl` | string | yes | Connection URL (use `{{username}}` and `{{password}}` as placeholders) |
| `forProvider.username` | string | yes | Username for Vault to authenticate to the database |
| `forProvider.password` | string | yes | Password for Vault to authenticate to the database |
| `forProvider.allowedRoles` | []string | no | Restrict which roles can use this connection |
| `forProvider.maxConnectionLifetime` | string | no | Maximum connection lifetime (e.g. `5m`) |
| `forProvider.maxIdleConnections` | int | no | Maximum idle connections |
| `forProvider.maxOpenConnections` | int | no | Maximum open connections |
| `forProvider.verifyConnection` | bool | no | Verify the connection on configuration |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example

```yaml
apiVersion: databasebackend.vault.m.crossplane.io/v1beta1
kind: DatabaseBackend
metadata:
  name: postgres-production
  namespace: vault
spec:
  forProvider:
    backend: database
    name: postgres-production
    pluginName: postgresql-database-plugin
    connectionUrl: "postgresql://{{username}}:{{password}}@postgres.example.com:5432/vault"
    username: vault_admin
    password: s3cret
    allowedRoles:
      - "readonly"
      - "readwrite"
    verifyConnection: true
  providerConfigRef:
    name: default
```
