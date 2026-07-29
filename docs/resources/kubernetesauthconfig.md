# KubernetesAuthConfig

**API Version**: `kubernetesauthconfig.vault.m.crossplane.io/v1beta1`

Configure the Kubernetes auth method with token reviewer settings and CA certificates.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | Mount path of the Kubernetes auth method |
| `forProvider.kubernetesHost` | string | yes | Kubernetes API server URL |
| `forProvider.kubernetesCaCert` | string | no | CA certificate for the Kubernetes API (PEM) |
| `forProvider.tokenReviewerJwt` | string | no | JWT from a service account with token review permission |
| `forProvider.issuer` | string | no | Kubernetes token issuer (default: `kubernetes/serviceaccount`) |
| `forProvider.disableIssValidation` | bool | no | Disable issuer validation |
| `forProvider.disableLocalCaJwt` | bool | no | Disable local CA JWT validation |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example

```yaml
apiVersion: kubernetesauthconfig.vault.m.crossplane.io/v1beta1
kind: KubernetesAuthConfig
metadata:
  name: k8s-infra
  namespace: vault
spec:
  forProvider:
    backend: k8s-infrastructure
    kubernetesHost: https://kubernetes.default.svc
    kubernetesCaCert: |
      -----BEGIN CERTIFICATE-----
      MIIB...
      -----END CERTIFICATE-----
    tokenReviewerJwt: eyJhbGciOiJSUzI1NiIsImtpZCI6...
    disableLocalCaJwt: false
  providerConfigRef:
    name: default
```

## Service Account Setup

The `tokenReviewerJwt` requires a Kubernetes service account with token review permissions:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: vault-token-reviewer
  namespace: vault
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: vault-token-reviewer
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
  - kind: ServiceAccount
    name: vault-token-reviewer
    namespace: vault
```

The JWT can be retrieved with:

```bash
kubectl create token vault-token-reviewer -n vault
```
