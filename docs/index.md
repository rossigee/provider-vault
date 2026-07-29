# Provider Vault Documentation

A Crossplane v2 provider for managing HashiCorp Vault resources with complete namespace isolation for multi-tenancy.

## Quick Links

- [Getting Started](getting-started.md) — Installation and first resource
- [ProviderConfig](configuration.md) — Authentication and connection setup
- [Development](development.md) — Building, testing, and contributing

## Resource Documentation

### Auth & Identity

| Resource | API Group | Description |
|----------|-----------|-------------|
| [AuthMethod](resources/authmethod.md) | `authmethod.vault.m.crossplane.io` | Enable, configure, tune, and disable auth methods |
| [AuthBackendRole](resources/authbackendrole.md) | `authbackendrole.vault.m.crossplane.io` | JWT/AppRole/Kubernetes auth roles for identity-to-policy mapping |
| [KubernetesAuthConfig](resources/kubernetesauthconfig.md) | `kubernetesauthconfig.vault.m.crossplane.io` | Kubernetes auth method configuration with token reviewers |
| [AppRoleSecretID](resources/approlesecretid.md) | `approlesecretid.vault.m.crossplane.io` | Generate and manage AppRole SecretIDs |

### Secrets & Encryption

| Resource | API Group | Description |
|----------|-----------|-------------|
| [KVSecret](resources/kvsecret.md) | `kvsecret.vault.m.crossplane.io` | KV v2 secret create/read/update/delete |
| [TransitKey](resources/transitkey.md) | `transitkey.vault.m.crossplane.io` | Encryption key management |
| [Token](resources/token.md) | `token.vault.m.crossplane.io` | Token create/renew/revoke with auto-renewal |

### PKI & Certificates

| Resource | API Group | Description |
|----------|-----------|-------------|
| [Mount](resources/mount.md) | `mount.vault.m.crossplane.io` | Secret engine mount enable/tune/disable |
| [PKIConfig](resources/pkiconfig.md) | `pkiconfig.vault.m.crossplane.io` | PKI root CA generation and URL configuration |
| [SecretBackendRole](resources/secretbackendrole.md) | `secretbackendrole.vault.m.crossplane.io` | PKI certificate role configuration |
| [Certificate](resources/certificate.md) | `certificate.vault.m.crossplane.io` | PKI certificate issuance with auto-renewal |

### Databases

| Resource | API Group | Description |
|----------|-----------|-------------|
| [DatabaseBackend](resources/databasebackend.md) | `databasebackend.vault.m.crossplane.io` | Database connection configuration |
| [DatabaseRole](resources/databaserole.md) | `databaserole.vault.m.crossplane.io` | Dynamic database credential roles |

### Access Control & Policies

| Resource | API Group | Description |
|----------|-----------|-------------|
| [Policy](resources/policy.md) | `policy.vault.m.crossplane.io` | ACL policy management |
| [IdentityEntity](resources/identityentity.md) | `identityentity.vault.m.crossplane.io` | Identity entity management |
| [IdentityGroup](resources/identitygroup.md) | `identitygroup.vault.m.crossplane.io` | Identity group management |

## Container Registry

- **Primary**: `ghcr.io/rossigee/provider-vault:v0.2.7`

## Repository

GitHub: [rossigee/provider-vault](https://github.com/rossigee/provider-vault)
