# Backup Plugin

> Automated PostgreSQL backups with scheduling and optional cloud upload. **Free, MIT licensed.**

## Install

```bash
nself plugin install backup
```

## What It Does

Runs scheduled `pg_dump` backups of your Postgres database with configurable retention policies. Supports uploading backups to S3-compatible storage (MinIO or any S3 provider). Free tier handles standard scheduling.

For encryption, multi-target storage (S3/R2/GCS/B2/MinIO simultaneously), and restore tracking, see the pro variant: [[plugin-backup-pro]].

## Free vs Pro

| Feature | Free (this) | Pro ([[plugin-backup-pro]]) |
|---------|-------------|------------------------------|
| Scheduled pg_dump | Yes | Yes |
| Local storage | Yes | Yes |
| S3-compatible upload | One target | Multi-target |
| AES-256 encryption at rest | No | Yes |
| Restore job tracking | No | Yes (idempotent resume) |
| Artifact checksums | No | Yes |
| Webhook events | No | Yes |
| Port | 3050 | 3210 |
| License required | No | Yes |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `BACKUP_PORT` | `3050` | Backup service port |
| `BACKUP_SCHEDULE` | `0 2 * * *` | Cron schedule (default: 2am daily) |
| `BACKUP_RETENTION_DAYS` | `7` | Days to keep backups |
| `BACKUP_S3_BUCKET` | — | S3/MinIO bucket name (optional) |
| `BACKUP_S3_ENDPOINT` | — | S3 endpoint URL (optional) |

## Ports

| Port | Purpose |
|------|---------|
| 3050 | Backup service REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_backup_jobs` — backup job history and status
- `np_backup_schedules` — configured backup schedules

## Nginx Routes

None; backup service is internal only.

## API

```
GET  /health           — Health check
GET  /backups          — List backup history
POST /backups/trigger  — Trigger immediate backup
POST /backups/restore  — Restore from backup
```

## See Also

- [[plugin-backup-pro]] — pro variant with encryption, multi-target, and restore tracking
- [[Plugins]] — full plugin index

← [[Plugins]] | [[Home]] →
