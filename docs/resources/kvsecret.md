# KVSecret

**API Version**: `kvsecret.vault.m.crossplane.io/v1beta1`

Create, read, update, and delete versioned KV v2 secrets.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.path` | string | yes | Secret path within the mount (e.g. `services/api/prod`) |
| `forProvider.data` | map[string]string | yes | Secret key-value pairs |
| `forProvider.mountPath` | string | yes | KV v2 engine mount path (e.g. `secret`) |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example

```yaml
apiVersion: kvsecret.vault.m.crossplane.io/v1beta1
kind: KVSecret
metadata:
  name: api-keys
  namespace: production
spec:
  forProvider:
    path: services/api/prod
    mountPath: secret
    data:
      api_key: sk-abc123
      db_password: s3cret
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Writes a new secret version at the specified path
- **Update**: Creates a new version of the existing secret (KV v2 maintains version history)
- **Delete**: Soft-deletes the latest version (can be undeleted from Vault)
- **Destroy**: Permanently deletes all versions (when deletion policy is set to `Delete`)
