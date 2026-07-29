# TransitKey

**API Version**: `transitkey.vault.m.crossplane.io/v1beta1`

Manage encryption keys in Vault's transit secrets engine.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | Transit engine mount path (e.g. `transit`) |
| `forProvider.name` | string | yes | Key name |
| `forProvider.type` | string | no | Key type: `aes128-gcm96`, `aes256-gcm96`, `chacha20-poly1305`, `ed25519`, `ecdsa-p256`, `rsa-2048`, `rsa-4096` (default: `aes256-gcm96`) |
| `forProvider.convergentEncryption` | bool | no | Enable convergent encryption |
| `forProvider.derived` | bool | no | Enable key derivation |
| `forProvider.exportable` | bool | no | Allow key export |
| `forProvider.allowPlaintextBackup` | bool | no | Allow plaintext backup of the key |
| `forProvider.autoRotatePeriod` | string | no | Auto-rotation period |
| `forProvider.minDecryptionVersion` | int | no | Minimum decryption version |
| `forProvider.minEncryptionVersion` | int | no | Minimum encryption version |

## Example

```yaml
apiVersion: transitkey.vault.m.crossplane.io/v1beta1
kind: TransitKey
metadata:
  name: app-encryption-key
  namespace: vault
spec:
  forProvider:
    backend: transit
    name: app-encryption-key
    type: aes256-gcm96
    convergentEncryption: false
    derived: false
    exportable: true
    minDecryptionVersion: 1
  providerConfigRef:
    name: default
```

## Key Types

| Type | Algorithm | Purpose |
|------|-----------|---------|
| `aes128-gcm96` | AES-128-GCM | Symmetric encryption |
| `aes256-gcm96` | AES-256-GCM | Symmetric encryption (default) |
| `chacha20-poly1305` | ChaCha20-Poly1305 | Symmetric encryption |
| `ed25519` | Ed25519 | Signing |
| `ecdsa-p256` | ECDSA P-256 | Signing |
| `rsa-2048` | RSA 2048-bit | Signing and encryption |
| `rsa-4096` | RSA 4096-bit | Signing and encryption |
