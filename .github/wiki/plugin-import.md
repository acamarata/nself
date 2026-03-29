> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# Import Plugin

> Data import service — CSV/JSON to Postgres with validation and error reporting. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install import
```

## What It Does

Imports CSV or JSON data into any Postgres table. Validates each row against the table schema before import, produces a detailed error report for invalid rows, and uses transactions to ensure atomicity. Supports field mapping (rename columns on import), default value injection, and deduplication by unique key.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `IMPORT_PORT` | `3053` | Import service port |
| `IMPORT_MAX_FILE_SIZE` | `100MB` | Maximum upload file size |
| `IMPORT_BATCH_SIZE` | `1000` | Rows per database transaction |
| `IMPORT_ERROR_THRESHOLD` | `0.05` | Abort if error rate exceeds this fraction |

## Ports

| Port | Purpose |
|------|---------|
| 3053 | Import REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_import_jobs` — import job queue and status
- `np_import_errors` — per-row validation errors

## Nginx Routes

| Route | Target |
|-------|--------|
| `/import/` | Import API |
| `/import/upload` | File upload endpoint |
