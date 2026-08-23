# Provider Vault

[![CI](https://img.shields.io/github/actions/workflow/status/rossigee/provider-vault/ci.yml?branch=master)][build]
[![Version](https://img.shields.io/github/v/release/rossigee/provider-vault)][releases]
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[build]: https://github.com/rossigee/provider-vault/actions/workflows/ci.yml
[releases]: https://github.com/rossigee/provider-vault/releases

A Crossplane v2 provider for managing HashiCorp Vault resources with complete namespace isolation for multi-tenancy.

## Container Registry

- **Primary**: `ghcr.io/rossigee/provider-vault:v0.2.7`

## Overview

A lightweight Crossplane v2 provider for managing HashiCorp Vault resources, designed as a native Kubernetes-native replacement for the Terraform-based [`upbound/provider-vault`](https://github.com/upbound/provider-vault). Uses raw HTTP client (no official Vault Go SDK dependency) to avoid transitive dependency conflicts and keep the provider slim.

## Features

- **Crossplane v2 Architecture**: Complete namespace-scoped resource management
- **Multi-tenancy**: All resources isolated by namespace for team separation
- **KV v2 Secrets**: Create, read, update, and delete versioned key-value secrets
- **ACL Policies**: Manage Vault ACL policies with full HCL support
- **Auth Methods**: Enable, configure, tune, and disable auth methods
- **Secret Engine Mounts**: Enable and configure secret engines (KV v2, PKI, etc.)
- **PKI Roles**: Configure certificate roles for PKI secret engines
- **PKI Config**: Generate root CA certificates for PKI secret engines
- **Certificate Issuance**: Issue TLS certificates from PKI roles with automatic renewal
- **Database Backends**: Configure database connections (PostgreSQL, MySQL, etc.)
- **Transit Keys**: Encryption key management for transit secret engine
- **Token Management**: Create, renew, and revoke Vault tokens with automatic renewal
- **Identity Entities**: Manage Vault identity entities with policies and metadata
- **Identity Groups**: Manage Vault identity groups with member entities
- **Auth Backend Roles**: Create JWT, AppRole, and Kubernetes auth roles for identity-to-policy mapping
- **AppRoleSecretID**: Generate and manage AppRole SecretIDs with automatic connection secret publishing
- **KubernetesAuthConfig**: Configure Kubernetes auth method with configurable token reviewers and TTLs
- **Token-based Auth**: Authenticate to Vault using periodic tokens via Kubernetes secrets
- **Custom CA Support**: SSL_CERT_FILE environment variable for internal CA certificates
- **Namespaced ProviderConfig**: Provider configuration scoped to namespaces for multi-tenant setups

## Getting Started

### Prerequisites

- Kubernetes cluster with Crossplane installed
- HashiCorp Vault server with token authentication
- Vault token with appropriate permissions

### Installation

```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-vault:v0.2.7
```

### Configuration

Create a secret with your Vault token:

```bash
kubectl create secret generic vault-credentials \
  --from-literal=credentials=your-vault-token-here \
  -n crossplane-system
```

If your Vault uses an internal CA, create a configmap with the CA certificate:

```bash
kubectl create configmap vault-ca \
  --from-file=ca.crt=/path/to/vault-ca.pem \
  -n crossplane-system
```

Create the ProviderConfig:

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
      key: credentials
```

## Usage

### Create a KV v2 Secret

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

### Create an ACL Policy

```yaml
apiVersion: policy.vault.m.crossplane.io/v1beta1
kind: Policy
metadata:
  name: app-reader
  namespace: production
spec:
  forProvider:
    name: app-reader
    policy: |
      path "secret/data/app/*" {
        capabilities = ["read", "list"]
      }
      path "secret/metadata/app/*" {
        capabilities = ["list"]
      }
  providerConfigRef:
    name: default
```

### Enable an Auth Method

```yaml
apiVersion: authmethod.vault.m.crossplane.io/v1beta1
kind: AuthMethod
metadata:
  name: github-actions
  namespace: production
spec:
  forProvider:
    mountPath: github-actions
    type: jwt
    config:
      oidc_discovery_url: https://token.actions.githubusercontent.com
      bound_issuer: https://token.actions.githubusercontent.com
  providerConfigRef:
    name: default
```

### Create a Secret Engine Mount

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

### Create a PKI Role

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
  providerConfigRef:
    name: default
```

### Generate an AppRole SecretID

```yaml
apiVersion: approlesecretid.vault.m.crossplane.io/v1beta1
kind: AppRoleSecretID
metadata:
  name: my-app-secretid
  namespace: production
spec:
  forProvider:
    backend: approle
    roleName: my-app
    metadata:
      created-by: crossplane
      environment: production
    cidrList:
      - "10.0.0.0/8"
  writeConnectionSecretToRef:
    name: my-app-secretid
  providerConfigRef:
    name: default
```

The `secret_id` and `secret_id_accessor` are automatically written to the connection secret for use by consuming applications.

### Create a JWT Auth Role

```yaml
apiVersion: authbackendrole.vault.m.crossplane.io/v1beta1
kind: AuthBackendRole
metadata:
  name: jwt-flux-system
  namespace: vault
spec:
  forProvider:
    backend: jwt-production
    roleName: flux-system
    roleType: jwt
    boundAudiences:
      - https://k8s-api.master.golder.lan
    boundSubject: "system:serviceaccount:flux-system:external-secrets"
    userClaim: sub
    tokenPolicies:
      - default
      - external-secrets-production
    tokenTtl: 3600
    tokenMaxTtl: 3600
  providerConfigRef:
    name: default
```

### Generate a PKI Root CA

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
  providerConfigRef:
    name: default
```

### Issue a Certificate

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

The Certificate resource automatically writes the issued certificate, CA chain, and private key to a Kubernetes Secret (specified via `writeConnectionSecretToRef`). The secret contains `tls.crt`, `ca.crt`, and `tls.key` keys.

## Resource Types

| Resource | API Version | Description |
|----------|-------------|-------------|
| KVSecret | `kvsecret.vault.m.crossplane.io/v1beta1` | KV v2 secret create/read/update/delete |
| Policy | `policy.vault.m.crossplane.io/v1beta1` | ACL policy management |
| AuthMethod | `authmethod.vault.m.crossplane.io/v1beta1` | Auth method enable/tune/disable |
| Mount | `mount.vault.m.crossplane.io/v1beta1` | Secret engine mount enable/tune/disable |
| SecretBackendRole | `secretbackendrole.vault.m.crossplane.io/v1beta1` | PKI certificate role configuration |
| AuthBackendRole | `authbackendrole.vault.m.crossplane.io/v1beta1` | JWT/AppRole/Kubernetes auth role management |
| PKIConfig | `pkiconfig.vault.m.crossplane.io/v1beta1` | PKI root CA generation and URL configuration |
| Certificate | `certificate.vault.m.crossplane.io/v1beta1` | PKI certificate issuance with auto-renewal |
| DatabaseBackend | `databasebackend.vault.m.crossplane.io/v1beta1` | Database connection configuration (PostgreSQL, MySQL, etc.) |
| DatabaseRole | `databaserole.vault.m.crossplane.io/v1beta1` | Dynamic database credential roles |
| TransitKey | `transitkey.vault.m.crossplane.io/v1beta1` | Encryption key management |
| Token | `token.vault.m.crossplane.io/v1beta1` | Token create/renew/revoke with k8s Secret output |
| IdentityEntity | `identityentity.vault.m.crossplane.io/v1beta1` | Identity entity management |
| IdentityGroup | `identitygroup.vault.m.crossplane.io/v1beta1` | Identity group management |
| AppRoleSecretID | `approlesecretid.vault.m.crossplane.io/v1beta1` | AppRole SecretID generation and management |
| KubernetesAuthConfig | `kubernetesauthconfig.vault.m.crossplane.io/v1beta1` | Kubernetes auth method configuration |
| ProviderConfig | `vault.m.crossplane.io/v1beta1` | Provider authentication and configuration |

## Unsupported Vault APIs

The following Vault APIs are not yet supported by this provider:

- AWS/Azure/GCP secrets engine
- Database secrets engine (role and credential configuration)
- AppRole role-id management
- JWT/OIDC auth configuration (beyond basic enable)
- AppRoleSecretID management (now supported — see above)
- Kubernetes auth configuration (now supported)
- LDAP auth configuration
- Token create/manage/renew
- Transit encryption key management
- AWS/Azure/GCP secrets engine configuration
- Audit device management
- Identity entities and groups
- Namespace management (Vault Enterprise)
- KV v1 (non-versioned) secrets
- Lease renew/revoke
- Rate limit quotas
- Mount tuning and configuration

## Development

```bash
# Build the provider
make build

# Run tests
make test

# Lint code
make lint

# Generate CRDs
make generate

# Build and publish
make publish VERSION=v0.2.7 PLATFORMS=linux_amd64
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

provider-vault is under the Apache 2.0 license.

## Implementation

This provider is a native Crossplane controller that directly implements the provider APIs without using Terraform or upjet scaffolding. This approach yields smaller binaries, simpler code, and reduced dependencies.
