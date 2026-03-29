> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# Audit Plugin

> Tamper-evident audit trail — log all API calls, data changes, and user actions. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install audit
```

## What It Does

Creates a tamper-evident audit log of all API requests, Hasura data mutations, and user actions in your nSelf stack. Each log entry is hash-chained to the previous entry so any tampering is detectable. Provides a query API and dashboard for audit review, compliance exports, and anomaly detection. Satisfies audit requirements for SOC 2, HIPAA, and PCI-DSS.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `AUDIT_PORT` | `3061` | Audit service port |
| `AUDIT_RETENTION_DAYS` | `2555` | Log retention (7 years default) |
| `AUDIT_HASH_ALGORITHM` | `sha256` | Hash algorithm for chain |
| `AUDIT_EXCLUDE_PATHS` | `/health` | Paths to exclude from logging |
| `AUDIT_PII_MASKING` | `true` | Mask PII fields in logs |

## Ports

| Port | Purpose |
|------|---------|
| 3061 | Audit log REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_audit_entries` — hash-chained audit log entries
- `np_audit_exports` — compliance export records

## Nginx Routes

| Route | Target |
|-------|--------|
| `/audit/` | Audit log query API |
