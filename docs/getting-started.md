# Getting Started

## Prerequisites

- Kubernetes cluster with Crossplane v2.0+ installed
- HashiCorp Vault server with token authentication
- Vault token with appropriate permissions for the resources you want to manage

## Installation

```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-vault:v0.2.7
```

Verify the provider is healthy:

```bash
kubectl get provider provider-vault
kubectl get providerrevisions
```

## Your First Resource

### 1. Create credentials and ProviderConfig

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

```bash
kubectl create secret generic vault-credentials \
  --from-literal=token=your-vault-token-here \
  -n crossplane-system
```

### 2. Create a KV v2 secret

```yaml
apiVersion: kvsecret.vault.m.crossplane.io/v1beta1
kind: KVSecret
metadata:
  name: app-secret
  namespace: production
spec:
  forProvider:
    path: services/api/prod
    mountPath: secret
    data:
      api_key: sk-abc123
  providerConfigRef:
    name: default
```

### 3. Verify

```bash
kubectl get kvsecret -n production
kubectl get managed
```

## Next Steps

- See the [ProviderConfig reference](configuration.md) for advanced configuration (internal CA, insecure mode, etc.)
- Browse [resource documentation](index.md) for specific resource types
- Check the [examples](../examples/) directory for complete YAML manifests
