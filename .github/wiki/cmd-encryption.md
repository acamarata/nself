# nself encryption

> Manage BYOK per-tenant encryption for nSelf Cloud (Enterprise tier only).

> **Enterprise tier required.** BYOK encryption requires `NSELF_BYOK=true` and a valid Enterprise license. This command has no effect on self-hosted Community or ɳSelf+ deployments.

## Synopsis

```bash
nself encryption <subcommand> [flags]
```

## Description

`nself encryption` manages Bring Your Own Key (BYOK) encryption for nSelf Cloud tenants. Each tenant supplies a Customer Managed Key (CMK) hosted in AWS KMS, GCP Cloud KMS, or HashiCorp Vault Transit. nSelf uses envelope encryption: data is encrypted with a Data Encryption Key (DEK), and the DEK is wrapped by the tenant's CMK. The CMK never leaves the tenant's KMS.

Key operations: configure a KMS provider, verify connectivity with a wrap/unwrap round-trip, rotate DEKs after a CMK rotation, check the current configuration status, and review the key event audit trail.

Set `BYOK_PLUGIN_URL` to point at your BYOK plugin endpoint. If unset, the command falls back to `NSELF_API_URL`, then `http://localhost:3741`. Set `NSELF_TENANT_ID` to scope requests to a specific tenant.

## Subcommands

| Subcommand | Description |
|-----------|-------------|
| `configure` | Configure a KMS provider for BYOK encryption |
| `verify` | Test KMS connectivity (wrap+unwrap round-trip) |
| `rotate` | Rotate data encryption keys |
| `status` | Show BYOK configuration and last verification |
| `key-events` | List the key event audit trail |

## Flags

### nself encryption configure

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--provider` | `–` | string | `""` | KMS provider: `aws`, `gcp`, or `vault` (required) |
| `--key-id` | `–` | string | `""` | AWS KMS key ARN or alias |
| `--key-name` | `–` | string | `""` | GCP Cloud KMS key resource path |
| `--key-path` | `–` | string | `""` | HashiCorp Vault Transit key path |
| `--endpoint` | `–` | string | `""` | Vault endpoint URL |
| `--region` | `–` | string | `""` | AWS or GCP region |
| `--credentials-ref` | `–` | string | `""` | `np_secrets` key name holding KMS credentials |

### nself encryption rotate

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dry-run` | `–` | bool | false | Estimate record counts without re-encrypting |

## Examples

```bash
# Configure AWS KMS
nself encryption configure \
  --provider aws \
  --key-id arn:aws:kms:us-east-1:123456789:key/abc123 \
  --region us-east-1
```

```bash
# Configure GCP Cloud KMS
nself encryption configure \
  --provider gcp \
  --key-name projects/my-project/locations/global/keyRings/my-ring/cryptoKeys/my-key
```

```bash
# Configure HashiCorp Vault Transit
nself encryption configure \
  --provider vault \
  --key-path transit/keys/tenant-abc \
  --endpoint https://vault.example.com
```

```bash
# Verify KMS connectivity after configuration
nself encryption verify
```

```bash
# Check current BYOK configuration
nself encryption status
```

```bash
# Preview a key rotation without applying it
nself encryption rotate --dry-run
```

```bash
# Rotate DEKs after rotating the CMK in your KMS
nself encryption rotate
```

```bash
# Review key event audit trail
nself encryption key-events
```

## See Also

- [cmd-license.md](cmd-license.md) — manage your nSelf license key (Enterprise tier required)
- [cmd-audit.md](cmd-audit.md) — view the security audit log
- [cmd-doctor.md](cmd-doctor.md) — verify environment and configuration health
- [cmd-config.md](cmd-config.md) — view and set nSelf configuration values
- [cmd-status.md](cmd-status.md) — view the running state of your nSelf install

← [[Commands]] | [[Home]] →
