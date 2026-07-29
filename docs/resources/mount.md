# Mount

**API Version**: `mount.vault.m.crossplane.io/v1beta1`

Enable, tune, and disable secret engine mounts (KV, PKI, Transit, Database, etc.).

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.path` | string | yes | Mount path (e.g. `kv`, `pki-production`) |
| `forProvider.type` | string | yes | Engine type (e.g. `kv-v2`, `pki`, `transit`, `database`) |
| `forProvider.description` | string | no | Human-readable description |
| `forProvider.defaultLeaseTtl` | int (seconds) | no | Default lease TTL |
| `forProvider.maxLeaseTtl` | int (seconds) | no | Maximum lease TTL |
| `forProvider.options` | map[string]string | no | Engine-specific options (e.g. `{"version": "2"}` for KV) |
| `forProvider.config` | map[string]string | no | Mount configuration options |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example (KV v2 Engine)

```yaml
apiVersion: mount.vault.m.crossplane.io/v1beta1
kind: Mount
metadata:
  name: kv-secrets
  namespace: vault
spec:
  forProvider:
    path: kv
    type: kv-v2
    description: Application secrets
    options:
      version: "2"
  providerConfigRef:
    name: default
```

## Example (PKI Engine)

```yaml
apiVersion: mount.vault.m.crossplane.io/v1beta1
kind: Mount
metadata:
  name: pki-production-ecdsa
  namespace: vault
spec:
  forProvider:
    path: pki-production-ecdsa
    type: pki
    description: Production ECDSA PKI engine
    maxLeaseTtl: 87600
  providerConfigRef:
    name: default
```

## Common Engine Types

| Type | Description |
|------|-------------|
| `kv` | KV v1 (non-versioned) |
| `kv-v2` | KV v2 (versioned) |
| `pki` | Public Key Infrastructure |
| `transit` | Encryption as a service |
| `database` | Dynamic database credentials |
| `aws` | AWS secrets engine |
| `azure` | Azure secrets engine |
| `gcp` | GCP secrets engine |
