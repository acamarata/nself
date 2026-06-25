# nself pitr

> Point-in-time recovery (PITR) operations: enable WAL archiving, create base backups, and restore to any point in your retention window.

## Synopsis

```bash
nself pitr <subcommand> [flags]
nself pitr enable
nself pitr disable
nself pitr status
nself pitr base-backup
nself pitr restore --to <timestamp>
```

## Description

`nself pitr` manages Postgres point-in-time recovery. PITR is built on WAL (Write-Ahead Log) archiving: Postgres ships WAL segments to a configured remote destination continuously; a base backup provides the starting point for any restore.

With PITR enabled you can restore your database to any second within the retention window, not just to a named backup snapshot.

Note: `nself db pitr` provides the same subcommands as part of the `db` command tree. `nself pitr` is the top-level alias. Both are fully supported.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `enable` | Enable WAL archiving and PITR infrastructure |
| `disable` | Disable WAL archiving (retains existing archives) |
| `status` | Show PITR status: archiving state, last WAL segment, retention |
| `base-backup` | Trigger a base backup to the configured WAL archive destination |
| `restore` | Restore to a specific timestamp from the WAL archive |

## Flags

### `pitr enable`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--destination` | string | — | WAL archive destination (e.g. `s3:bucket/wal`) |
| `--retention` | string | `7d` | WAL archive retention window |
| `--dry-run` | bool | false | Show what would change without enabling |

### `pitr status`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | bool | false | Emit status as JSON |

### `pitr base-backup`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--label` | string | auto | Base backup label |
| `--compress` | bool | true | Compress the base backup |

### `pitr restore`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--to` | string | — | Target timestamp (ISO-8601: `2026-01-15T02:30:00Z`) |
| `--dry-run` | bool | false | Validate restore path without applying |
| `--yes` | bool | false | Skip confirmation prompt |

## Examples

```bash
# Enable PITR with S3 WAL archiving
nself pitr enable --destination s3:my-bucket/wal
```

```bash
# Check PITR status
nself pitr status
```

```bash
# Create a base backup before a major migration
nself pitr base-backup --label pre-migration-2026-01-15
```

```bash
# Restore to a specific second (staging only; not for production)
nself pitr restore --to 2026-01-15T02:30:00Z
```

```bash
# Preview what a restore would do without applying
nself pitr restore --to 2026-01-15T02:30:00Z --dry-run
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (archiving not configured, destination unreachable, invalid timestamp) |
| 2 | PITR not enabled — run `nself pitr enable` first |

## Safety Notes

- Run `nself pitr base-backup` on staging, never on production, unless you have confirmed a DR scenario.
- `nself pitr restore` requires a database stop. Use `--dry-run` first to verify the archive is intact and the target timestamp is reachable.
- WAL archiving adds I/O. For projects under 10 GB total data, the overhead is negligible. For larger projects, tune `--retention` to control archive storage cost.

## See Also

- [[cmd-backup.md]] — scheduled full/streaming backups
- [[cmd-dr.md]] — full disaster-recovery workflows
- [[cmd-db.md]] — database operations including `db pitr`
- [[Guide-Backup-Restore.md]] — backup and restore how-to guide

← [[Commands]] | [[Home]] →
