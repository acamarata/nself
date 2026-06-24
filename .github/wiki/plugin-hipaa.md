# Plugin: hipaa

HIPAA compliance add-on for ɳSelf-backed applications. Provides a PHI column registry,
a tamper-evident PHI access audit log with enforced 6-year retention, de-identification
helpers (masking, tokenization, redaction), BAA workflow tracking, and an encryption
audit endpoint.

**Requires:** ɳSelf+ license (`pro` entitlement).

---

## Port Conflict Warning

> **Port 3212 is assigned to both `hipaa` and `admin-api`.** Running both on the same
> host will cause a bind failure. Reassign one before starting both:
>
> ```bash
> # In hipaa .env
> HIPAA_PLUGIN_PORT=3213
> # OR in admin-api .env
> ADMIN_API_PORT=3213
> ```
>
> This conflict is tracked in F10-PORT-REGISTRY.md. A port reassignment task is filed
> separately — until resolved, operators must manually configure one plugin to use an
> alternate port.

---

## Installation

```bash
nself plugin install hipaa
nself plugin start hipaa
```

The plugin starts on port 3212 (configurable via `HIPAA_PLUGIN_PORT`).

---

## PHI Column Registry

Register any table column that holds Protected Health Information (PHI). The registry
drives automatic de-identification and access logging.

### Register a PHI column

```bash
curl -X POST http://localhost:3212/hipaa/phi-columns \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Source-Account-ID: primary" \
  -H "Content-Type: application/json" \
  -d '{
    "table_name": "patients",
    "column_name": "ssn",
    "phi_category": "ssn",
    "de_id_method": "tokenize",
    "audit_note": "Required for billing"
  }'
```

### List registered PHI columns

```bash
curl http://localhost:3212/hipaa/phi-columns \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Source-Account-ID: primary"
```

### Unregister a PHI column

```bash
curl -X DELETE http://localhost:3212/hipaa/phi-columns/{id} \
  -H "Authorization: Bearer $TOKEN"
```

### PHI Categories

| Category | Masked form |
|---|---|
| `ssn` | `XXX-XX-6789` (last 4 kept) |
| `dob` | `1985-XX-XX` (year only) |
| `name` | `J*** D**` |
| `mrn` | `XXXXX1234` (last 4 kept) |
| `phone` | `XXX-XXX-4321` |
| `email` | `j***e@example.com` |
| `address` | `123 [STREET MASKED]` |
| `other` | First + last char preserved |

### De-identification methods

| Method | Behaviour |
|---|---|
| `mask` | Replaces most of the value with X or * using the category masker |
| `tokenize` | Replaces value with an opaque `tok_...` token; detokenizable by `phi:detokenize` role |
| `redact` | Replaces value with `[REDACTED]` — not reversible |

---

## PHI Access Audit Log

Every access to a registered PHI column creates an immutable audit row.
Retention is enforced at the database level via a generated column:

```sql
retain_until DATE GENERATED ALWAYS AS ((accessed_at + INTERVAL '6 years')::DATE) STORED
```

Rows cannot be deleted before `retain_until` (enforced by RLS in migration 002).

### Query audit log

```bash
curl "http://localhost:3212/hipaa/audit-log" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-HIPAA-Role: phi:read" \
  -H "X-Source-Account-ID: primary"
```

Optional query parameters: `from`, `to` (ISO 8601 dates), `accessor_id`, `limit` (max 1000).

### Export to CSV

```bash
curl "http://localhost:3212/hipaa/audit-log/export" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-HIPAA-Role: phi:admin" > audit-export.csv
```

### Log a PHI access manually

Applications can append entries directly (in addition to the Hasura Event Trigger webhook):

```bash
curl -X POST http://localhost:3212/hipaa/audit-log \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "table_name": "patients",
    "column_names": ["ssn", "dob"],
    "row_count": 3,
    "accessor_email": "clinician@clinic.org",
    "purpose": "treatment"
  }'
```

Valid `purpose` values: `treatment`, `payment`, `operations`, `other`.

---

## De-identification Endpoint

Apply Safe Harbor de-identification to a batch of rows. All registered PHI columns are
processed according to their `de_id_method`.

```bash
curl -X POST http://localhost:3212/hipaa/deidentify \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "table_name": "patients",
    "rows": [
      {"id": 1, "name": "Jane Smith", "ssn": "123-45-6789", "dob": "1990-03-14"}
    ]
  }'
```

Response:

```json
{
  "table_name": "patients",
  "rows_processed": 1,
  "rows": [
    {"id": 1, "name": "J*** S****", "ssn": "XXX-XX-6789", "dob": "1990-XX-XX"}
  ]
}
```

---

## Tokenization

```bash
# Tokenize a PHI value
curl -X POST http://localhost:3212/hipaa/tokenize \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"value": "Jane Smith"}'
# → {"token": "tok_a3b2c4..."}

# Detokenize (requires X-HIPAA-Role: phi:detokenize)
curl http://localhost:3212/hipaa/tokenize/tok_a3b2c4... \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-HIPAA-Role: phi:detokenize"
```

---

## BAA Workflow

Track Business Associate Agreement signing:

```bash
# Request a BAA
curl -X POST http://localhost:3212/hipaa/baa/request \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"signed_by": "Business Partner Inc", "signer_email": "legal@partner.org"}'

# List BAAs
curl http://localhost:3212/hipaa/baa -H "Authorization: Bearer $TOKEN"

# Activate after signing
curl -X POST http://localhost:3212/hipaa/baa/activate \
  -H "Authorization: Bearer $TOKEN" -d '{"id": "<baa-uuid>"}'
```

---

## Encryption Audit

```bash
curl http://localhost:3212/hipaa/encryption-audit \
  -H "Authorization: Bearer $TOKEN"
```

Returns pgcrypto availability, TLS-in-transit status, and Vault Transit connectivity
(if `NSELF_HIPAA_VAULT=true`).

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string (must use `sslmode=require`) |
| `NSELF_HIPAA` | No | `true` | Enable PHI registry + access logging |
| `NSELF_HIPAA_BAA` | No | `false` | Enable BAA workflow |
| `NSELF_HIPAA_VAULT` | No | `false` | Use HashiCorp Vault Transit for tokenization |
| `NSELF_HIPAA_VAULT_ADDR` | No | — | Vault server address |
| `NSELF_HIPAA_VAULT_TOKEN` | No | — | Vault token (store in `.env.secrets`, never committed) |
| `NSELF_HIPAA_RETENTION_YEARS` | No | `6` | Must be >= 6 (HIPAA minimum) |
| `NSELF_HIPAA_BAA_BUCKET` | No | `baa-documents` | MinIO bucket for BAA PDFs |
| `HIPAA_PLUGIN_PORT` | No | `3212` | HTTP port — **change if admin-api occupies 3212** |
| `HIPAA_PLUGIN_HOST` | No | `0.0.0.0` | Bind address |
| `HIPAA_API_KEY` | No | — | Plugin-to-plugin auth key |
| `HIPAA_LOG_LEVEL` | No | `info` | Log verbosity (`debug`, `info`, `warn`, `error`) |

---

## Database Tables

| Table | Purpose |
|---|---|
| `np_phi_columns` | PHI column registry; unique per `(source_account_id, table_name, column_name)` |
| `np_phi_audit_log` | Immutable PHI access log; `retain_until` generated at insert (access + 6 years) |
| `np_baa_records` | BAA signing lifecycle records |

All tables use `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation.

---

## HIPAA Roles

| Role header value | Permitted operations |
|---|---|
| `phi:read` | Read audit log |
| `phi:admin` | Read + export audit log; manage PHI columns |
| `phi:detokenize` | Detokenize PHI tokens |
| `hipaa:admin` | All of the above |

Pass the role in the `X-HIPAA-Role` request header.

---

## Docker

```bash
docker pull nself-org/plugin-hipaa:latest
docker run \
  -e DATABASE_URL=postgres://... \
  -e NSELF_HIPAA=true \
  -p 3212:3212 \
  nself-org/plugin-hipaa:latest
```

---

## Related

- [Plugin-Install](Plugin-Install.md)
- [Plugin-Licensing](Plugin-Licensing.md)
- [Plugin-Catalog](Plugin-Catalog.md)
