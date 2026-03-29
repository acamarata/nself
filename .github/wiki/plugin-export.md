> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# Export Plugin

> Data export service — table to CSV, JSON, and XLSX with async job queue. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install export
```

## What It Does

Exports data from any Postgres table to CSV, JSON, or XLSX format. Large exports run as background jobs to avoid request timeouts. Results are stored temporarily in MinIO or local storage and delivered via download link. Supports column filtering, row filtering with SQL WHERE clauses, and per-tenant data isolation.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `EXPORT_PORT` | `3049` | Export service port |
| `EXPORT_MAX_ROWS` | `1000000` | Max rows per export job |
| `EXPORT_LINK_TTL` | `3600` | Download link expiry in seconds |
| `EXPORT_STORAGE` | `local` | Storage backend: `local` or `minio` |

## Ports

| Port | Purpose |
|------|---------|
| 3049 | Export REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_export_jobs` — export job queue and status
- `np_export_templates` — saved export definitions

## Nginx Routes

| Route | Target |
|-------|--------|
| `/export/` | Export API |
| `/export/download/{token}` | Secure file download |
