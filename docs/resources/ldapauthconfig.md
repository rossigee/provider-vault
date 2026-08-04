# LDAPAuthConfig

**API Version**: `ldapauthconfig.vault.m.crossplane.io/v1beta1`

Configure LDAP authentication method for Vault.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.backend` | string | yes | Auth mount path (e.g., `ldap`) |
| `forProvider.url` | string | yes | LDAP URL (e.g., `ldaps://ldap.example.com:636`) |
| `forProvider.userdn` | string | no | User DN template (e.g., `cn={{.Username}},ou=users,dc=example,dc=com`) |
| `forProvider.groupdn` | string | no | Group DN to search in |
| `forProvider.userattr` | string | no | Username attribute (default: `cn`) |
| `forProvider.groupattr` | string | no | Group membership attribute (default: `member`) |
| `forProvider.binddn` | string | no | DN for bind user |
| `forProvider.bindpass` | string | no | Password for bind user |
| `forProvider.starttls` | bool | no | Use StartTLS |
| `providerConfigRef.name` | string | yes | ProviderConfig name |

## Example (LDAPS)

```yaml
apiVersion: ldapauthconfig.vault.m.crossplane.io/v1beta1
kind: LDAPAuthConfig
metadata:
  name: ldap
  namespace: vault
spec:
  forProvider:
    backend: ldap
    url: ldaps://ldap.example.com:636
    userdn: cn={{.Username}},ou=users,dc=example,dc=com
    groupdn: ou=groups,dc=example,dc=com
    userattr: cn
    groupattr: member
    binddn: cn=admin,dc=example,dc=com
    bindpass: secret
  providerConfigRef:
    name: default
```

## Example (LDAP with StartTLS)

```yaml
apiVersion: ldapauthconfig.vault.m.crossplane.io/v1beta1
kind: LDAPAuthConfig
metadata:
  name: ldap-starttls
  namespace: vault
spec:
  forProvider:
    backend: ldap
    url: ldap://ldap.example.com:389
    userdn: uid={{.Username}},ou=people,dc=example,dc=com
    groupdn: ou=groups,dc=example,dc=com
    starttls: true
  providerConfigRef:
    name: default
```