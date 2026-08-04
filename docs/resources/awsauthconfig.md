# AWSAuthConfig

**API Version**: `awsauthconfig.vault.m.crossplane.io/v1beta1`

Configure AWS IAM authentication method for Vault.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | Auth mount path (e.g., `aws`) |
| `forProvider.iamServerIdHeader` | string | no | Header value for server ID validation |
| `forProvider.stsEndpoint` | string | no | Custom STS endpoint |
| `forProvider.stsRegion` | string | no | AWS region for STS |
| `forProvider.iamEndpoint` | string | no | Custom IAM endpoint |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example (IAM Auth)

```yaml
apiVersion: awsauthconfig.vault.m.crossplane.io/v1beta1
kind: AWSAuthConfig
metadata:
  name: aws
  namespace: vault
spec:
  forProvider:
    backend: aws
    stsRegion: us-east-1
  providerConfigRef:
    name: default
```

## Example (Custom Endpoints)

```yaml
apiVersion: awsauthconfig.vault.m.crossplane.io/v1beta1
kind: AWSAuthConfig
metadata:
  name: aws-govcloud
  namespace: vault
spec:
  forProvider:
    backend: aws-govcloud
    iamEndpoint: https://iam.govcloud.amazonaws.com
    stsEndpoint: https://sts.govcloud.amazonaws.com
    stsRegion: us-gov-west-1
    iamServerIdHeader: vault.example.com
  providerConfigRef:
    name: default
```