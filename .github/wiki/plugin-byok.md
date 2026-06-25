# BYOK Plugin

> Bring Your Own Key per-tenant encryption with customer-managed keys (CMK). **Max-tier Pro plugin.**

> **Requires:** the ɳSelf+ (max) license tier. Set it with `nself license set nself_max_...` before installing.

## Install

```bash
nself license set nself_max_xxxxx...
nself plugin install byok
```

The plugin listens on port **3743** (per F10-PORT-REGISTRY; paypal uses 3741).

## What It Does

BYOK lets each tenant own its encryption key. Application data is sealed with AES-256-GCM, and the data key is wrapped by a customer-managed key in the tenant's own AWS KMS, GCP Cloud KMS, or HashiCorp Vault Transit. nSelf never holds an unwrappable root key, which is what HIPAA, FedRAMP High, FFIEC, and DORA require for key control.

- **Envelope encryption:** a fresh 256-bit DEK encrypts each record group with AES-256-GCM (random 12-byte nonce, no reuse); the DEK is wrapped by the tenant's CMK and stored beside the ciphertext.
- **Tenant isolation:** a record sealed under tenant A's CMK cannot be decrypted with tenant B's CMK — the wrong DEK fails GCM authentication rather than returning plaintext.
- **Revocation-safe:** if a CMK is revoked or disabled, unwrap fails and the plugin surfaces an error; it never falls back to a cached plaintext DEK.
- **Key rotation:** re-encrypts a tenant's records under fresh DEKs while reusing the CMK; rotate the CMK in your KMS and update the key reference to change roots.

## AWS IAM Policy for CMK Access

The nSelf workload principal needs the minimum policy on the tenant CMK:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["kms:Encrypt", "kms:Decrypt", "kms:DescribeKey"],
      "Resource": "arn:aws:kms:<region>:<account>:key/<key-id>"
    }
  ]
}
```

`kms:Encrypt` wraps DEKs, `kms:Decrypt` unwraps them, `kms:DescribeKey` reads key state. GCP Cloud KMS and Vault Transit follow the same minimal-grant model.

## Feature Flags

| Flag | Default | Description |
|------|---------|-------------|
| `NSELF_BYOK` | `false` | Enable BYOK customer-managed key encryption. |
| `NSELF_BYOK_DEK_CACHE` | `false` | Cache decrypted DEKs in memory to reduce KMS calls. |
| `NSELF_BYOK_MULTI_REGION` | `false` | Enable multi-region KMS replication support. |

## Tables

- `np_kms_configs` — per-tenant KMS provider and CMK reference.
- `np_encrypted_values` — ciphertext, wrapped DEK, IV, CMK reference.
- `np_encryption_key_events` — rotation and key-state audit trail.

## Compliance

Satisfies the customer-key-control requirements of HIPAA, FedRAMP High, FFIEC, and DORA. Enterprise (max-tier) only.
