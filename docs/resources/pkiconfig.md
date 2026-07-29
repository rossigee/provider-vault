# PKIConfig

**API Version**: `pkiconfig.vault.m.crossplane.io/v1beta1`

Generate root CA certificates and configure PKI engine URLs (CRL, issuing certificates, OCSP).

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | PKI engine mount path (e.g. `pki-production-ecdsa`) |
| `forProvider.type` | string | yes | CA type: `root_internal` or `root_exported` |
| `forProvider.commonName` | string | yes | Common Name for the root CA |
| `forProvider.ttl` | string | no | Root CA TTL (e.g. `87600h` for 10 years) |
| `forProvider.keyType` | string | no | Key type: `rsa` or `ec` |
| `forProvider.keyBits` | int | no | Key bits (e.g. `384` for ECDSA, `4096` for RSA) |
| `forProvider.organization` | []string | no | Organization (O) |
| `forProvider.ou` | []string | no | Organizational Unit (OU) |
| `forProvider.country` | []string | no | Country (C) |
| `forProvider.locality` | []string | no | Locality (L) |
| `forProvider.province` | []string | no | Province (ST) |
| `forProvider.streetAddress` | []string | no | Street address |
| `forProvider.postalCode` | []string | no | Postal code |
| `forProvider.permittedDnsDomains` | []string | no | Permitted DNS domains |
| `forProvider.maxPathLength` | int | no | Max path length for intermediate CAs |
| `forProvider.excludeCnFromSans` | bool | no | Exclude CN from SANs |
| `forProvider.issuingCertificates` | []string | no | Issuing certificates URLs |
| `forProvider.crlDistributionPoints` | []string | no | CRL distribution point URLs |
| `forProvider.ocspServers` | []string | no | OCSP responder URLs |

## Example

```yaml
apiVersion: pkiconfig.vault.m.crossplane.io/v1beta1
kind: PKIConfig
metadata:
  name: pki-production-ecdsa
  namespace: vault
spec:
  forProvider:
    backend: pki-production-ecdsa
    type: root_internal
    commonName: Production ECDSA Root CA
    ttl: 87600h
    keyType: ec
    keyBits: 384
    organization:
      - MyOrg
    issuingCertificates:
      - https://vault.example.com/v1/pki-production-ecdsa/ca
    crlDistributionPoints:
      - https://vault.example.com/v1/pki-production-ecdsa/crl
  providerConfigRef:
    name: default
```

## Behavior

- **root_internal**: Private key stays in Vault (never exported)
- **root_exported**: Private key is returned in the response and written to the connection secret
- Once generated, the CA is permanent — the resource becomes immutable
