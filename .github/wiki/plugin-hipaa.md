# Plugin: HIPAA

**Port:** 3212 (⚠️ see Port Conflict below)
**Bundle:** Unbundled — standalone install
**License:** ɳSelf+ required (`NSELF_HIPAA=true`)

The HIPAA plugin provides HIPAA-compliant PHI management tooling: a column registry, 6-year-retention audit log, and Safe Harbor de-identification.

## Installation

```bash
nself plugin install hipaa
```

## PHI Column Registration

Register which columns in your application tables contain Protected Health Information:

```bash
curl -X POST http://localhost:3212/hipaa/phi-columns \
  -H "X-Source-Account-ID: primary" \
  -H "X-HIPAA-Role: phi:admin" \
  -d '{"table_name":"patients","column_name":"ssn","phi_category":"ssn","de_id_method":"mask"}'
```

`phi_category` values: `name`, `dob`, `ssn`, `mrn`, `address`, `phone`, `email`, `other`
`de_id_method` values: `mask`, `tokenize`, `redact`

## PHI Access Log

Query the audit trail:

```bash
# List recent PHI accesses
curl http://localhost:3212/hipaa/audit-log \
  -H "X-Source-Account-ID: primary" \
  -H "X-HIPAA-Role: phi:read"

# Filter by accessor
curl "http://localhost:3212/hipaa/audit-log?accessor_id=uuid&from=2025-01-01" \
  -H "X-HIPAA-Role: phi:read"

# Export as CSV for auditors
curl http://localhost:3212/hipaa/audit-log/export \
  -H "X-HIPAA-Role: phi:admin" > phi-audit.csv
```

**Retention:** Every entry includes `retain_until` (6 years from `accessed_at`). The Postgres RLS policy blocks `DELETE` until `retain_until < CURRENT_DATE`, implementing 45 CFR § 164.530(j) at the database layer.

## De-Identification (Safe Harbor)

De-identify a batch of rows from a registered table:

```bash
curl -X POST http://localhost:3212/hipaa/deidentify \
  -d '{
    "table_name": "patients",
    "rows": [{"ssn":"123-45-6789","dob":"1985-03-14","name":"John Doe"}]
  }'
```

Response:

```json
{
  "table_name": "patients",
  "rows_processed": 1,
  "rows": [{"ssn":"XXX-XX-6789","dob":"1985-XX-XX","name":"J*** D**"}]
}
```

Tokenize a single PHI value:

```bash
curl -X POST http://localhost:3212/hipaa/tokenize -d '{"value":"123-45-6789"}'
# → {"token":"tok_a1b2c3d4..."}
```

## HIPAA Audit Report Generation

Generate an audit report for a date range:

```bash
# Get all PHI accesses in Q1 2025
curl "http://localhost:3212/hipaa/audit-log?from=2025-01-01&to=2025-03-31&limit=1000" \
  -H "X-HIPAA-Role: phi:admin" | jq '.[] | {accessor: .accessor_email, table: .table_name, at: .accessed_at}'
```

## Environment Variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | Postgres connection (required) |
| `NSELF_HIPAA` | `true` to enable PHI endpoints |
| `NSELF_HIPAA_BAA` | `true` to enable BAA management |
| `HIPAA_API_KEY` | Shared secret for `X-HIPAA-API-Key` header auth |
| `HIPAA_PLUGIN_PORT` | Override listen port (default 3212) |

## ⚠️ Port Conflict — 3212

Port 3212 is also claimed by the `admin-api` plugin. If both plugins are installed on the same nSelf instance, reconfigure one:

```bash
# Set hipaa to use an alternate port
nself plugin config hipaa HIPAA_PLUGIN_PORT=3213
```

This conflict is tracked in F10-PORT-REGISTRY.md for resolution in a future registry cleanup task.

## Roles

| `X-HIPAA-Role` header | Access |
|---|---|
| `phi:read` | Read audit log |
| `phi:admin` | Read + export + unregister PHI columns |
| `phi:detokenize` | Detokenize stored tokens |
| `hipaa:admin` | All of the above + post-retention log purge |
