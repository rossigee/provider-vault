# GCPAuthConfig

**API Version**: `gcpauthconfig.vault.m.crossplane.io/v1beta1`

Configure GCP GCE authentication method for Vault.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | Auth mount path (e.g., `gcp`) |
| `forProvider.projectId` | string | yes | GCP project ID |
| `forProvider.serviceAccountEmail` | string | yes | Service account email |
| `forProvider.instanceGroupNames` | string | no | Compute instance group names |
| `forProvider.zone` | string | no | GCP zone |
| `forProvider.region` | string | no | GCP region |
| `forProvider.clusterName` | string | no | GKE cluster name |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example (GCE)

```yaml
apiVersion: gcpauthconfig.vault.m.crossplane.io/v1beta1
kind: GCPAuthConfig
metadata:
  name: gcp
  namespace: vault
spec:
  forProvider:
    backend: gcp
    projectId: my-project
    serviceAccountEmail: vault-auth@my-project.iam.gserviceaccount.com
    zone: us-central1-a
  providerConfigRef:
    name: default
```

## Example (GKE)

```yaml
apiVersion: gcpauthconfig.vault.m.crossplane.io/v1beta1
kind: GCPAuthConfig
metadata:
  name: gcp-gke
  namespace: vault
spec:
  forProvider:
    backend: gcp
    projectId: my-gke-project
    serviceAccountEmail: vault-gke@my-gke-project.iam.gserviceaccount.com
    region: us-central1
    clusterName: vault-cluster
  providerConfigRef:
    name: default
```