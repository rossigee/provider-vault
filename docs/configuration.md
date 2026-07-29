# ProviderConfig

The ProviderConfig connects the Crossplane provider to your Vault server.

## API Version

`vault.m.crossplane.io/v1beta1`

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `address` | string | yes | Vault server URL (e.g. `https://vault.example.com:8200`) |
| `insecure` | bool | no | Skip TLS verification (default: `false`) |
| `credentials.source` | string | yes | Must be `Secret` |
| `credentials.secretRef.name` | string | yes | Name of the Kubernetes Secret containing the Vault token |
| `credentials.secretRef.namespace` | string | yes | Namespace of the secret |
| `credentials.secretRef.key` | string | yes | Key in the Kubernetes Secret. The secret value must be a JSON object with a `token` field (e.g. `{"token":"hvs.xxx"}`), or a raw token string |
| `vaultNamespace` | string | no | Vault Enterprise namespace for all API requests (sent as `X-Vault-Namespace` header) |
| `tls.caCertSecretRef` | object | no | Reference to a Kubernetes Secret containing the CA certificate (key: `ca.crt`) |
| `tls.clientCertSecretRef` | object | no | Reference to a Kubernetes Secret containing TLS client cert and key (keys: `tls.crt`, `tls.key`) |

## Basic Example

```yaml
apiVersion: vault.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
  namespace: crossplane-system
spec:
  address: https://vault.example.com:8200
  insecure: false
  credentials:
    source: Secret
    secretRef:
      name: vault-credentials
      namespace: crossplane-system
      key: token
```

## TLS Configuration

If your Vault server uses an internal CA or requires mutual TLS (client certificates):

### CA Certificate

```yaml
spec:
  tls:
    caCertSecretRef:
      name: vault-ca          # Secret containing ca.crt key
      namespace: crossplane-system
      key: ca.crt
```

### Mutual TLS (Client Certificate)

```yaml
spec:
  tls:
    caCertSecretRef:
      name: vault-ca
      namespace: crossplane-system
      key: ca.crt
    clientCertSecretRef:
      name: vault-tls-client  # Secret containing tls.crt and tls.key
      namespace: crossplane-system
      key: tls.crt
```

The TLS client secret must use the standard Kubernetes TLS secret format:
- `tls.crt` — PEM-encoded client certificate
- `tls.key` — PEM-encoded client private key

## Vault Enterprise Namespace

For Vault Enterprise clusters with namespaces enabled:

```yaml
spec:
  address: https://vault.example.com
  vaultNamespace: admin
  credentials:
    source: Secret
    secretRef:
      name: vault-credentials
      namespace: crossplane-system
      key: credentials
```

This sends the `X-Vault-Namespace: admin` header with every API request.

## Insecure Mode

For development only:

```yaml
spec:
  address: https://vault.dev.example.com:8200
  insecure: true
```

## Namespaced ProviderConfigs

ProviderConfigs can be scoped to a specific namespace for multi-tenant setups:

```yaml
apiVersion: vault.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: team-a
  namespace: team-a-ns
spec:
  address: https://vault-team-a.example.com:8200
  credentials:
    source: Secret
    secretRef:
      name: vault-team-a-token
      namespace: team-a-ns
      key: token
```

## Token Secret Format

The token secret must contain the key specified in `secretRef.key`. The value can be either:

### JSON Wrapper (recommended)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: vault-credentials
  namespace: crossplane-system
type: Opaque
stringData:
  credentials: '{"token":"hvs.your-vault-token-here"}'
```

### Raw Token

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: vault-credentials
  namespace: crossplane-system
type: Opaque
stringData:
  token: hvs.your-vault-token-here
```

The provider automatically handles both formats by attempting to parse the value as JSON first, then falling back to using the raw string.
