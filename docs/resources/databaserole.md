# DatabaseRole

**API Version**: `databaserole.vault.m.crossplane.io/v1beta1`

Create dynamic database credential roles that generate temporary database users on demand.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | Mount path of the database engine |
| `forProvider.name` | string | yes | Name of the role |
| `forProvider.dbName` | string | yes | Name of the database connection configuration |
| `forProvider.creationStatements` | []string | no | SQL statements to create the database user |
| `forProvider.revocationStatements` | []string | no | SQL statements to revoke/drop the user |
| `forProvider.rollbackStatements` | []string | no | SQL statements to roll back a failed creation |
| `forProvider.renewStatements` | []string | no | SQL statements to renew a user |
| `forProvider.defaultTtl` | string | no | Default TTL for generated credentials |
| `forProvider.maxTtl` | string | no | Maximum TTL for generated credentials |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example

```yaml
apiVersion: databaserole.vault.m.crossplane.io/v1beta1
kind: DatabaseRole
metadata:
  name: app-reader
  namespace: vault
spec:
  forProvider:
    backend: database
    name: app-reader
    dbName: postgres-production
    creationStatements:
      - "CREATE USER \"{{name}}\" WITH PASSWORD '{{password}}' VALID UNTIL '{{expiration}}';"
      - "GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";"
    defaultTtl: 1h
    maxTtl: 24h
  providerConfigRef:
    name: default
```

## Common Creation Statements

### PostgreSQL Read-Only
```sql
CREATE USER "{{name}}" WITH PASSWORD '{{password}}' VALID UNTIL '{{expiration}}';
GRANT SELECT ON ALL TABLES IN SCHEMA public TO "{{name}}";
```

### PostgreSQL Read-Write
```sql
CREATE USER "{{name}}" WITH PASSWORD '{{password}}' VALID UNTIL '{{expiration}}';
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO "{{name}}";
```

### MySQL
```sql
CREATE USER '{{name}}'@'%' IDENTIFIED BY '{{password}}';
GRANT SELECT ON myapp.* TO '{{name}}'@'%';
```
