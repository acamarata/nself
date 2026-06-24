# Plugin: byok

**Tier:** max (Enterprise only) · **Port:** 3741 · **Category:** compliance

Per-tenant envelope encryption using customer-managed keys (CMK). Tenants supply their own AWS KMS, GCP Cloud KMS, or HashiCorp Vault Transit key. Satisfies HIPAA, FedRAMP High, FFIEC, and DORA key-control requirements. nSelf generates a fresh 256-bit DEK per record-group, encrypts with AES-256-GCM, wraps the DEK with the tenant's CMK, and stores only the ciphertext and wrapped DEK — never the plaintext key material.

> **Max-tier requirement:** `byok` requires the `max` (Enterprise) license. It will not install or start on `pro` or `free` tier deployments. Run `nself license info` to confirm tier before installing.

## Install

```bash
nself plugin install byok
nself build
nself start
```

## Port Note

> **Port conflict — 3741:** The `byok` plugin defaults to port **3741**. The `paypal` plugin also claims port 3741 in some registry configurations. If you run both plugins on the same host, set `BYOK_PLUGIN_PORT` to an available port (e.g. `3742`) before starting byok. The port registry conflict between byok and paypal is tracked in `F10-PORT-REGISTRY.md` for resolution in a dedicated registry task.

To override the port:

```bash
export BYOK_PLUGIN_PORT=3742
nself build
nself start
```

## AES-256-GCM Envelope Scheme

```
Record plaintext
      │
      ├── Generate 256-bit DEK (crypto/rand)
      │         │                       │
      │         ▼                       ▼
      │   AES-256-GCM encrypt    WrapKey(DEK, CMK)
      │   + 12-byte nonce        via KMS provider
      │         │                       │
      └─────────┴────────────────────────┘
                    EncryptedBundle persisted:
                    ciphertext | encrypted_dek | iv | kms_key_ref
```

DEK is zeroed from memory immediately after use. The plaintext DEK is never written to disk or logs.

## Provider Setup

### AWS KMS

Grant the nSelf IAM role these actions on the CMK:

- `kms:GenerateDataKey`
- `kms:Decrypt`
- `kms:DescribeKey`

### GCP Cloud KMS

Grant the nSelf service account `roles/cloudkms.cryptoKeyEncrypterDecrypter` on the key.

### HashiCorp Vault Transit

Enable `transit` secret engine, create an `aes256-gcm96` key, and issue a token with `encrypt` + `decrypt` + `keys:read` capabilities. Store the token in `np_secrets` and reference it via `credentials_ref`.

Full IAM policy examples: `plugins-pro/paid/byok/README.md`.

## API Endpoints

All endpoints require `Authorization: Bearer <nSelf JWT>` and `X-Tenant-ID`.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/encryption/kms` | Get current KMS config for tenant |
| `POST` | `/encryption/kms` | Configure KMS provider |
| `PUT` | `/encryption/kms` | Update KMS config |
| `POST` | `/encryption/kms/verify` | Verify KMS connectivity (live wrap/unwrap test) |
| `POST` | `/encryption/rotate` | Start key rotation job |
| `GET` | `/encryption/rotate/{job_id}` | Poll rotation job status |
| `GET` | `/encryption/key-events` | Key event audit trail |
| `GET` | `/health` | Plugin health check |
| `GET` | `/ready` | Readiness probe |

## Schema

| Table | Purpose |
|---|---|
| `np_kms_configs` | Per-tenant KMS provider, key ref, region, credentials ref |
| `np_encrypted_values` | AES-256-GCM bundles (ciphertext, wrapped DEK, IV, key ref) |
| `np_encryption_key_events` | Immutable audit trail (configured, verified, rotated, revoked) |

RLS policies enforce `tenant_id = X-Hasura-Tenant-Id` on all three tables. Cross-tenant access is blocked at the database level.

## Tenant Isolation

Each tenant's KMS config is isolated by `tenant_id`. Attempting to decrypt a tenant A bundle using tenant B's CMK fails with a GCM authentication tag error — the wrong DEK produces incorrect padding and the ciphertext is rejected. This is verified in the plugin's tenant isolation test suite (`go/crypto/tenant_isolation_test.go`).

## Key Rotation

`POST /encryption/rotate` queues a background job that:

1. Lists all `np_encrypted_values` for the tenant in batches (`NSELF_BYOK_ROTATE_BATCH_SIZE`, default 1000).
2. Unwraps each DEK with the current CMK.
3. Re-wraps with the new CMK version.
4. Updates `np_encrypted_values.encrypted_dek` in place.
5. Records a `key.rotated` event in `np_encryption_key_events`.

Poll `GET /encryption/rotate/{job_id}` for progress. Rotation does not decrypt or re-encrypt the data ciphertext — only the wrapped DEK is rotated.

## CMK Revocation

Revoking a CMK in AWS/GCP/Vault immediately blocks all `UnwrapKey` calls for that tenant. Decrypt operations return an error (no stale plaintext is served). This is the intended isolation property — the tenant's data becomes inaccessible until the CMK is restored or a new key is configured.

## Environment Variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | Postgres connection |
| `BYOK_PLUGIN_PORT` | No | `3741` | HTTP listener port |
| `NSELF_BYOK` | No | `false` | Enable BYOK feature |
| `NSELF_BYOK_DEK_CACHE` | No | `false` | Cache decrypted DEKs in memory |
| `NSELF_BYOK_DEK_CACHE_TTL` | No | `0` | DEK cache TTL (seconds; 0 = session) |
| `NSELF_BYOK_ROTATE_BATCH_SIZE` | No | `1000` | Rotation batch size |
| `NSELF_BYOK_ROTATE_RATE_LIMIT` | No | `100` | Rotation ops/sec |
| `NSELF_BYOK_MULTI_REGION` | No | `false` | Multi-region KMS replication |
| `NSELF_LICENSE_KEY` | No | — | License key override |

## Docker

```
docker pull nself-org/plugin-byok:latest
```

Multi-arch: `linux/amd64`, `linux/arm64`, `darwin/arm64`.

## See Also

- `plugins-pro/paid/byok/README.md` — full setup guide including IAM policies
- `plugin-compliance.md` — compliance plugin (free)
- `plugin-audit-log.md` — append-only audit log (free)
- `F10-PORT-REGISTRY.md` — port registry (byok/paypal 3741 conflict tracked here)
