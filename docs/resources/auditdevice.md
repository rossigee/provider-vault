# AuditDevice

`AuditDevice` enables and configures audit devices in Vault. Audit devices maintain a record of all requests and responses in Vault.

## Specification

```yaml
apiVersion: auditdevice.vault.m.crossplane.io/v1beta1
kind: AuditDevice
metadata:
  name: file-audit
spec:
  forProvider:
    path: file-audit
    type: file
    description: "File-based audit device"
    options:
      file_path: /var/log/vault/audit.log
    local: false
  providerConfigRef:
    name: vault-provider
```

## Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Path where the audit device is enabled |
| `type` | string | Yes | Type of audit device (file, socket, syslog, etc.) |
| `description` | string | No | Human-readable description |
| `options` | map[string]string | No | Device-specific options |
| `local` | bool | No | Whether this is a local mount (non-replicated) |

## Supported Device Types

- **file**: Writes audit logs to a file
- **socket**: Sends audit logs to a TCP/UDP socket
- **syslog**: Writes audit logs to the system logger
- **http**: Sends audit logs to an HTTP endpoint

## Examples

### File Audit Device

```yaml
apiVersion: auditdevice.vault.m.crossplane.io/v1beta1
kind: AuditDevice
metadata:
  name: file-audit
spec:
  forProvider:
    path: file-audit
    type: file
    description: "Main audit log"
    options:
      file_path: /var/log/vault/audit.log
      rotate_duration: 24h
      rotate_max_bytes: 104857600
  providerConfigRef:
    name: vault-provider
```

### HTTP Audit Device

```yaml
apiVersion: auditdevice.vault.m.crossplane.io/v1beta1
kind: AuditDevice
metadata:
  name: http-audit
spec:
  forProvider:
    path: http-audit
    type: http
    description: "HTTP audit webhook"
    options:
      address: https://audit.example.com/webhook
      format: json
  providerConfigRef:
    name: vault-provider
```

## Notes

- Deleting the AuditDevice resource will disable the audit device in Vault
- Some options are device-specific and may not apply to all types
- Local audit devices are not replicated to HA clusters