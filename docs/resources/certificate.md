# Certificate

**API Version**: `certificate.vault.m.crossplane.io/v1beta1`

Issue TLS certificates from a PKI role with automatic renewal. The certificate, private key, and CA chain are written to a Kubernetes Secret.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | PKI engine mount path (e.g. `pki-production-ecdsa`) |
| `forProvider.role` | string | yes | PKI role name to issue from |
| `forProvider.commonName` | string | yes | Common Name for the certificate |
| `forProvider.altNames` | []string | no | DNS subject alternative names |
| `forProvider.ipSans` | []string | no | IP subject alternative names |
| `forProvider.uriSans` | []string | no | URI subject alternative names |
| `forProvider.otherSans` | []string | no | Custom OID subject alternative names |
| `forProvider.ttl` | string | no | Certificate TTL (e.g. `2160h`). Defaults to role TTL |
| `forProvider.format` | string | no | Output format: `pem`, `der`, `pem_bundle` (default: `pem`) |
| `forProvider.privateKeyFormat` | string | no | Private key format: `der`, `pkcs8` (default: `der`) |
| `forProvider.renewBefore` | float | no | Renew when this fraction of TTL remains (default: `0.33`, i.e. 33%) |
| `writeConnectionSecretToRef.name` | string | yes | Name of the secret to write TLS data to |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Connection Secret

| Key | Description |
|-----|-------------|
| `tls.crt` | Issued certificate (PEM) |
| `tls.key` | Private key |
| `ca.crt` | Issuing CA certificate |
| `ca_chain` | Full CA chain |

## Example

```yaml
apiVersion: certificate.vault.m.crossplane.io/v1beta1
kind: Certificate
metadata:
  name: myapp-cert
  namespace: production
spec:
  forProvider:
    backend: pki-production-ecdsa
    role: cert-manager
    commonName: app.production.example.com
    altNames:
      - app.production.example.com
    ttl: 2160h
    renewBefore: 0.33
  writeConnectionSecretToRef:
    name: myapp-tls
  providerConfigRef:
    name: default
```

## Auto-Renewal

The Certificate resource automatically renews when the certificate age exceeds `(1 - renewBefore) * ttl`. For example, with a 90-day TTL and `renewBefore: 0.33`, renewal triggers at 60 days (67% of TTL elapsed).

On renewal, the connection secret is updated with the new certificate and private key.
