> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# Backup Pro Plugin

> Advanced PostgreSQL backup — multi-destination, encryption, and point-in-time recovery. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install backup-pro
```

## What It Does

Extends the free `backup` plugin with production-grade capabilities: AES-256 encryption at rest, simultaneous upload to multiple destinations (S3, GCS, Azure Blob, R2), point-in-time recovery (PITR) via WAL archiving, and backup verification by test-restoring to an isolated container. Includes a web dashboard for backup status and restore operations.

## Note on Free Tier

The free `backup` plugin handles standard scheduled pg_dump. This pro version adds encryption, multi-destination, PITR, and verification.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `BACKUP_PRO_PORT` | `3213` | Backup pro service port |
| `BACKUP_PRO_ENCRYPTION_KEY` | *(auto-generated)* | AES-256 encryption key |
| `BACKUP_PRO_DESTINATIONS` | `s3` | Comma-separated: `s3,gcs,azure,r2` |
| `BACKUP_PRO_PITR_ENABLED` | `false` | Enable WAL archiving for PITR |
| `BACKUP_PRO_VERIFY_ENABLED` | `true` | Verify backups via test restore |
| `BACKUP_PRO_RETENTION_DAYS` | `30` | Backup retention period |

## Ports

| Port | Purpose |
|------|---------|
| 3213 | Backup pro REST API and dashboard |

## Database Tables

4 tables added to your Postgres database:
- `np_backup_pro_jobs` — backup job history
- `np_backup_pro_destinations` — configured storage destinations
- `np_backup_pro_wal_segments` — WAL segment archive index (for PITR)
- `np_backup_pro_verifications` — test restore verification log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/backup/dashboard` | Backup status dashboard UI |
