# Hasura Metadata Backup

Daily export of Hasura metadata to a JSON file on disk.

## What it does

`nself backup hasura-metadata` calls the Hasura `export_metadata` API and writes the result to:

```
<backup_dir>/hasura-metadata-<YYYY-MM-DD>.json
```

The file is mode `0600`. The backup directory defaults to `./backups` (relative to the project directory) or the value of `NSELF_BACKUP_DIR`.

## Why it matters

Hasura metadata contains all tracked tables, relationships, permissions, remote schemas, and event triggers. Losing it means manually re-applying every schema change. A daily JSON snapshot makes recovery a single `nself db hasura metadata apply` call.

## Running it manually

```bash
nself backup hasura-metadata
```

Example output:

```
hasura metadata backup written to ./backups/hasura-metadata-2026-04-30.json
```

## Daily cron

Install the system-level cron (fires at 02:00 UTC):

```bash
# Linux (systemd) or macOS (launchd) — requires root
sudo nself maintenance hasura-metadata-cron --install
```

Check the status:

```bash
nself maintenance hasura-metadata-cron --status
```

Remove the cron:

```bash
sudo nself maintenance hasura-metadata-cron --remove
```

On Linux the timer is named `nself-hasura-metadata-backup.timer` (systemd).
On macOS the LaunchDaemon label is `org.nself.hasura-metadata-backup`.

Logs:

```bash
# Linux
journalctl -u nself-hasura-metadata-backup

# macOS
tail -f /var/log/nself-hasura-metadata-backup.log
```

## Doctor check: BACKUP-METADATA-01

`nself doctor --deep` includes the check `Hasura metadata backup (BACKUP-METADATA-01)`.

| Result | Meaning |
| ------ | ------- |
| pass | A `hasura-metadata-*.json` file exists and is less than 36 hours old |
| fail | No metadata backup found, or the most recent is older than 36 hours |
| warn | Backup directory does not exist |

Fix a fail result:

```bash
nself backup hasura-metadata
```

## Restore from backup

```bash
# Copy the backup JSON into the project's hasura/ directory.
cp backups/hasura-metadata-2026-04-30.json hasura/metadata.json

# Apply it to a running Hasura instance.
nself db hasura metadata apply
```

## Backup directory configuration

Set `NSELF_BACKUP_DIR` in `.env.local` or `.env.secrets` to override the default `./backups` path:

```bash
NSELF_BACKUP_DIR=/var/backups/nself
```

## Related pages

- [[cmd-backup]] — full backup command reference
- [[operations/release-cascade]] — release checklist that includes a metadata backup step
- [[Home]]
