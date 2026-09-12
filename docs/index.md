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
| [AuthMethod](resources/authmethod.md) | `authmethod.vault.m.crossplane.io/v1beta1` | Enable, configure, tune, and disable auth methods |
| [AuthBackendRole](resources/authbackendrole.md) | `authbackendrole.vault.m.crossplane.io/v1beta1` | JWT/AppRole/Kubernetes auth roles for identity-to-policy mapping |
| [KubernetesAuthConfig](resources/kubernetesauthconfig.md) | `kubernetesauthconfig.vault.m.crossplane.io/v1beta1` | Kubernetes auth method configuration with token reviewers |
| [AppRoleSecretID](resources/approlesecretid.md) | `approlesecretid.vault.m.crossplane.io/v1beta1` | Generate and manage AppRole SecretIDs |
| [JWTAuthConfig](resources/jwtauthconfig.md) | `jwtauthconfig.vault.m.crossplane.io/v1beta1` | JWT/OIDC auth method configuration |
| [LDAPAuthConfig](resources/ldapauthconfig.md) | `ldapauthconfig.vault.m.crossplane.io/v1beta1` | LDAP auth method configuration |
| [AWSAuthConfig](resources/awsauthconfig.md) | `awsauthconfig.vault.m.crossplane.io/v1beta1` | AWS IAM auth method configuration |
| [AzureAuthConfig](resources/azureauthconfig.md) | `azureauthconfig.vault.m.crossplane.io/v1beta1` | Azure MSI auth method configuration |
| [GCPAuthConfig](resources/gcpauthconfig.md) | `gcpauthconfig.vault.m.crossplane.io/v1beta1` | GCP GCE/GKE auth method configuration |

### Secrets & Encryption

| Resource | API Group | Description |
|----------|-----------|-------------|
| [KVSecret](resources/kvsecret.md) | `kvsecret.vault.m.crossplane.io/v1beta1` | KV v2 secret create/read/update/delete |
| [TransitKey](resources/transitkey.md) | `transitkey.vault.m.crossplane.io/v1beta1` | Encryption key management |
| [Token](resources/token.md) | `token.vault.m.crossplane.io/v1beta1` | Token create/renew/revoke with auto-renewal |

### PKI & Certificates

| Resource | API Group | Description |
|----------|-----------|-------------|
| [Mount](resources/mount.md) | `mount.vault.m.crossplane.io/v1beta1` | Secret engine mount enable/tune/disable |
| [PKIConfig](resources/pkiconfig.md) | `pkiconfig.vault.m.crossplane.io/v1beta1` | PKI root CA generation and URL configuration |
| [SecretBackendRole](resources/secretbackendrole.md) | `secretbackendrole.vault.m.crossplane.io/v1beta1` | PKI certificate role configuration |
| [Certificate](resources/certificate.md) | `certificate.vault.m.crossplane.io/v1beta1` | PKI certificate issuance with auto-renewal |

### Databases

| Resource | API Group | Description |
|----------|-----------|-------------|
| [DatabaseBackend](resources/databasebackend.md) | `databasebackend.vault.m.crossplane.io/v1beta1` | Database connection configuration |
| [DatabaseRole](resources/databaserole.md) | `databaserole.vault.m.crossplane.io/v1beta1` | Dynamic database credential roles |

### Access Control & Policies

| Resource | API Group | Description |
|----------|-----------|-------------|
| [Policy](resources/policy.md) | `policy.vault.m.crossplane.io/v1beta1` | ACL policy management |
| [IdentityEntity](resources/identityentity.md) | `identityentity.vault.m.crossplane.io/v1beta1` | Identity entity management |
| [IdentityGroup](resources/identitygroup.md) | `identitygroup.vault.m.crossplane.io/v1beta1` | Identity group management |
| [AuditDevice](resources/auditdevice.md) | `auditdevice.vault.m.crossplane.io/v1beta1` | Audit device configuration |

### Isolation & Governance

| Resource | API Group | Description |
|----------|-----------|-------------|
| [VaultNamespace](resources/namespaces.md) | `namespaces.vault.m.crossplane.io/v1beta1` | Vault namespace management |
| [Quota](resources/quota.md) | `quota.vault.m.crossplane.io/v1beta1` | Rate and lease quota limits |
| [LeaseRenewal](resources/leaserenewal.md) | `leaserenewal.vault.m.crossplane.io/v1beta1` | Automatic lease renewal |

## Container Registry

- **Primary**: `ghcr.io/rossigee/provider-vault:v0.2.31`

## API Coverage Gaps

Vault API surface not yet modeled by this provider:

- **Secrets engines**: AWS/Azure/GCP secrets engine credential brokering (config resources exist but not the engine-specific operations), SSH/Transform/KMIP secrets engines
- **Database secrets engine**: role and credential configuration (config resources exist)
- **AppRole**: role-id management
- **SSH/Transform/KMIP** secrets engines
- **Replication/DR** operation endpoints
- **Plugin catalog** management
- **Replication/DR** operation endpoints
- **Token**: create/renew/revoke (supported), but token roles and accessors not fully modeled
- **Lease**: renew/revoke not modeled as resources

## Repository

GitHub: [rossigee/provider-vault](https://github.com/rossigee/provider-vault)
