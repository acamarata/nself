# Backup Plugin

> PostgreSQL backup automation with cron scheduling, multi-target storage, and AES-256 encryption. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|----------------------|
| Free | $0 | $0 | No |
| Any bundle | $0.99/mo | $9.99/yr | If in bundle |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** Basic (this is a `tier: pro` plugin per F07-PRICING-TIERS).

## Bundle membership

Not currently bundled — purchase a tier subscription (Basic and up) for access.

Or get all bundles + all apps via **ɳSelf+** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install backup
nself build
```

The license is validated against `ping.nself.org/license/validate`. Tier is checked server-side; insufficient tier returns an error.

## Description

The backup pro plugin provides PostgreSQL backup and restore automation. It uses `pg_dump` to create consistent database snapshots on a configurable cron schedule, then uploads artifacts to any S3-compatible storage target (AWS S3, Cloudflare R2, MinIO, Backblaze B2, or GCS).

This is the **pro variant**, distinct from the free `backup` plugin (port 3050). The free plugin handles basic scheduled pg_dump to local storage only. The pro variant adds:

- **Cron scheduling** — any cron expression, e.g., `0 2 * * *` for 2am daily or `0 */6 * * *` for every 6 hours
- **Multi-target upload** — send artifacts to S3, R2, MinIO, B2, and GCS simultaneously
- **AES-256-GCM encryption** — encrypt artifacts before upload; set `BACKUP_ENCRYPTION_KEY` for at-rest protection
- **Configurable retention** — per-schedule retention policies with automatic cleanup via `/v1/cleanup`
- **Idempotent restore tracking** — failed restores resume from the last checkpoint via `np_backup_restore_jobs`
- **Artifact checksums** — SHA-256 hash stored per artifact in `np_backup_artifacts` for integrity verification
- **Webhook events** — `np_backup_webhook_events` table for external notifications on backup/restore state changes
- **Concurrency control** — `BACKUP_MAX_CONCURRENT` limits parallel backup jobs to prevent I/O saturation

Restore jobs are tracked so a failed restore can resume without re-pulling the full artifact.

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string (auto-provided by nSelf CLI) |
| `BACKUP_PLUGIN_PORT` | No | `3210` | Service port (distinct from free plugin's 3050) |
| `BACKUP_STORAGE_PATH` | No | `/tmp/nself-backups` | Local staging path before cloud upload |
| `BACKUP_S3_ENDPOINT` | No | — | S3-compatible endpoint URL (MinIO: `http://host:9000`, R2: `https://<acct>.r2.cloudflarestorage.com`) |
| `BACKUP_S3_BUCKET` | No | — | Bucket name for artifact upload |
| `BACKUP_S3_ACCESS_KEY` | No | — | Access key for the storage provider |
| `BACKUP_S3_SECRET_KEY` | No | — | Secret key for the storage provider |
| `BACKUP_S3_REGION` | No | `us-east-1` | Storage region (GCS: use `auto`) |
| `BACKUP_ENCRYPTION_KEY` | No | — | AES-256-GCM key; omit to store artifacts unencrypted |
| `BACKUP_DEFAULT_RETENTION_DAYS` | No | `30` | Days to keep artifacts before cleanup |
| `BACKUP_MAX_CONCURRENT` | No | `2` | Max parallel backup jobs |
| `BACKUP_PG_DUMP_PATH` | No | `pg_dump` | Path to pg_dump binary if not in PATH |
| `BACKUP_API_KEY` | No | — | Inter-plugin API key (auto-provided by nSelf CLI) |
| `BACKUP_RATE_LIMIT_MAX` | No | `100` | Max requests per rate limit window |
| `BACKUP_RATE_LIMIT_WINDOW_MS` | No | `60000` | Rate limit window in milliseconds |

Reference vault credentials. Never hardcode secrets in your `.env` file.

## Ports

| Port | Purpose |
|------|---------|
| 3210 | Backup service REST API |

Note: the free `backup` plugin runs on port 3050. This pro plugin uses 3210, so both can run side-by-side in the same nSelf deployment without port conflicts.

## Database Schema

Tables created (prefix `np_`):

| Table | Purpose |
|-------|---------|
| `np_backup_schedules` | Configured cron-based backup schedules |
| `np_backup_artifacts` | Backup artifact records with checksums and storage refs |
| `np_backup_restore_jobs` | Restore job state for idempotent resume |
| `np_backup_webhook_events` | Webhook event log for external notifications |

All tables use `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation.

## REST API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness probe |
| GET | `/` | Plugin capability list |
| GET | `/v1/status` | Backup status and statistics |
| POST | `/v1/backup` | Trigger immediate backup run |
| POST | `/v1/restore` | Start restore job from an artifact |
| GET | `/v1/restore/{id}` | Check restore job status |
| GET | `/{id}` | Get artifact metadata |
| GET | `/{id}/download` | Download artifact file |
| DELETE | `/{id}` | Delete artifact |
| POST | `/v1/cleanup` | Purge expired artifacts per retention policy |

## Examples

Create a schedule (daily at 2am, 30-day retention, with S3 upload):

```bash
nself plugin action backup create-schedule \
  --cron "0 2 * * *" \
  --retention-days 30 \
  --storage s3
```

Run an immediate backup:

```bash
nself plugin action backup run-backup --schedule-id sch_xxx
```

List artifacts:

```bash
nself plugin action backup list-backups
```

Restore from an artifact:

```bash
nself plugin action backup restore --artifact-id art_xxx
```

Check status via REST:

```bash
curl -H 'Authorization: Bearer $TOKEN' https://api.example.com/backup/v1/status
```

## Docker Hub

```bash
docker pull nself-org/plugin-backup:latest
```

Tag `latest` and `pro` are equivalent. The image includes `pg_dump` and `pg_restore` from the `postgresql-client` package.

## Source

Source-available (license required to run): [`plugins-pro/paid/backup/`](https://github.com/nself-org/plugins-pro/tree/main/paid/backup)

Note: `plugins-pro` is a private repository. Source access is granted to ɳSelf+ subscribers and Enterprise customers.

## See Also

- [[plugin-backup]] — free backup plugin (port 3050, basic scheduling, no encryption)
- [[Pricing]] — tier comparison
- [[Plugins]] — full plugin index

← [[Plugins]] | [[Home]] →
