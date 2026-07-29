# SecretBackendRole

**API Version**: `secretbackendrole.vault.m.crossplane.io/v1beta1`

Configure PKI certificate roles that define certificate issuance parameters (allowed domains, key types, TTLs, etc.).

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | PKI engine mount path |
| `forProvider.name` | string | yes | Role name |
| `forProvider.allowedDomains` | []string | no | Allowed DNS domains |
| `forProvider.allowSubdomains` | bool | no | Allow subdomains of allowed domains |
| `forProvider.allowBareDomains` | bool | no | Allow bare domains without subdomain |
| `forProvider.allowGlobDomains` | bool | no | Allow glob pattern matching of domains |
| `forProvider.allowAnyName` | bool | no | Allow any common name (dangerous) |
| `forProvider.keyType` | string | no | Key type: `rsa` or `ec` |
| `forProvider.keyBits` | int | no | Key bits (e.g. `384` for EC, `4096` for RSA) |
| `forProvider.signatureBits` | int | no | Signature bits |
| `forProvider.ttl` | string | no | Default TTL for issued certificates |
| `forProvider.maxTtl` | string | no | Maximum TTL for issued certificates |
| `forProvider.generateLease` | bool | no | Generate lease for certificates |
| `forProvider.enforceHostnames` | bool | no | Enforce valid hostnames |
| `forProvider.allowIpSans` | bool | no | Allow IP SANs |
| `forProvider.allowLocalhostFlag` | bool | no | Allow localhost |
| `forProvider.allowWildcardCertificates` | bool | no | Allow wildcard certificates |
| `forProvider.serverFlag` | bool | no | Set server usage flag |
| `forProvider.clientFlag` | bool | no | Set client usage flag |
| `forProvider.organization` | []string | no | Organization (O) |
| `forProvider.ou` | []string | no | Organizational Unit (OU) |
| `forProvider.country` | []string | no | Country (C) |
| `forProvider.locality` | []string | no | Locality (L) |
| `forProvider.province` | []string | no | Province (ST) |
| `forProvider.streetAddress` | []string | no | Street address |
| `forProvider.postalCode` | []string | no | Postal code |
| `forProvider.noStore` | bool | no | Do not store certificate in Vault |
| `forProvider.requireCn` | bool | no | Require Common Name |
| `forProvider.allowedOtherSans` | []string | no | Allowed custom OID SANs |
| `forProvider.allowedSerialNumbers` | []string | no | Allowed serial numbers |

## Example (ECDS)

```yaml
apiVersion: secretbackendrole.vault.m.crossplane.io/v1beta1
kind: SecretBackendRole
metadata:
  name: cert-manager-ecdsa
  namespace: vault
spec:
  forProvider:
    backend: pki-production-ecdsa
    name: cert-manager
    allowedDomains:
      - "*.svc.cluster.local"
    allowSubdomains: true
    allowBareDomains: true
    keyType: ec
    keyBits: 384
    ttl: 86400
    maxTtl: 604800
    serverFlag: true
    clientFlag: true
  providerConfigRef:
    name: default
```

## Example (RSA)

```yaml
apiVersion: secretbackendrole.vault.m.crossplane.io/v1beta1
kind: SecretBackendRole
metadata:
  name: cert-manager-rsa
  namespace: vault
spec:
  forProvider:
    backend: pki-production-rsa
    name: cert-manager
    allowedDomains:
      - "*.example.com"
    allowSubdomains: true
    allowBareDomains: true
    keyType: rsa
    keyBits: 4096
    ttl: 86400
    maxTtl: 604800
  providerConfigRef:
    name: default
```
