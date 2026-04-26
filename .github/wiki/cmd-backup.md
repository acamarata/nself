# nself backup

> Backup operations: create, list, restore, verify, prune, config, status, init-key.

## Synopsis

```
nself backup <subcommand> [flags] [args]
```

## Description

`nself backup` covers the full life cycle of project backups: full Postgres dumps, write-ahead-log (WAL) checkpoints, MinIO object snapshots, configuration metadata, and combined backups. Backups can be encrypted with age (one keypair per project) and pruned by retention policy.

Subcommands include `create` (one-shot full or incremental), `list` (filter by remote, environment, age), `restore` (from any backup ID or `latest`, with partial restore options), `verify` (checksum or full restore-test in a disposable container), and `prune` (apply daily/weekly/monthly retention).

Note: Point-in-time recovery (PITR) is not supported in v1.0.9. The `--point-in-time` flag has been removed. pgbackrest-based WAL replay is planned for v1.1.0.

`backup config --install-cron` installs systemd timers for full backups, WAL checkpoints, prune cycles, and weekly verify drills so an operator can hand off to automation.

## Subcommands

| Name | Description |
|------|-------------|
| `create` | Create a new backup |
| `list` | List available backups |
| `restore <backup-id\|latest>` | Restore from a backup |
| `verify <backup-id\|latest>` | Verify backup integrity |
| `prune` | Remove old backups by retention policy |
| `config` | View or install backup configuration (systemd timers) |
| `status` | Show backup subsystem status |
| `init-key` | Generate age encryption keypair for backups |

## Flags

### `backup create`

| Flag | Default | Description |
|------|---------|-------------|
| `--type` | `full` | Backup type: `full`, `wal`, `metadata`, `minio`, `all` |
| `--remote` | `""` | Remote name override |
| `--encrypt` | false | Force encryption on |
| `--no-encrypt` | false | Force encryption off |
| `--tag` | `""` | Human label for this backup |
| `--dry-run` | false | Preview only |

### `backup list`

| Flag | Default | Description |
|------|---------|-------------|
| `--remote` | `""` | Filter by remote name |
| `--env` | `""` | Filter by environment |
| `--since` | `""` | Show backups newer than duration (e.g. `24h`, `7d`) |
| `--format` | `table` | Output format: `table` or `json` |

### `backup restore <backup-id\|latest>`

| Flag | Default | Description |
|------|---------|-------------|
| `--to` | `""` | Restore to alternate directory |
| `--only` | `""` | Restore subset: `pg`, `minio`, `metadata` (comma-separated) |
| `--decrypt-key` | `""` | Path to age identity file |
| `--yes` | false | Skip confirmation |

Note: `--point-in-time` (PITR) is not available in v1.0.9 and has been removed. See v1.1.0 release notes for pgbackrest PITR support.

### `backup verify <backup-id\|latest>`

| Flag | Default | Description |
|------|---------|-------------|
| `--restore-test` | false | Spin up test container and restore |
| `--cleanup` | true | Remove test container after verify |
| `--keep` | false | Keep test container for inspection |

#### `--restore-test` deep restore verification

When `--restore-test` is set, `backup verify` goes beyond the default checksum check:

1. Spins up an ephemeral Postgres container on a random local port.
2. Restores the target `.dump` file via `pg_restore` into the container.
3. Runs the full smoke-query catalog (5+ system tables): user table count, live tuple count, `auth.users` presence, `hdb_catalog.hdb_metadata` presence, and `np_claw_conversations` presence. A zero user-table count triggers an explicit "row count mismatch, schema-only restore detected" failure.
4. Runs a sentinel CRUD round-trip: creates a temp schema, inserts a sentinel row, reads it back, and drops the schema. Completes in under 2 seconds. Cleaned up in defer even on panic.
5. Reports pass/fail with JSON output suitable for cron consumption.
6. Tears down the container and volume via `defer`, cleanup happens even on failure.

Exit code 0 = restore verified. Non-zero = verify failed. JSON output:

```json
{"backup_id":"latest","verified":true,"started_at":"2026-04-20T02:00:01Z","duration":"47.3s","method":"restore-test","details":"file size: 104857600 bytes"}
```

Use `--keep` to inspect the restored database before teardown (useful for debugging silent data corruption).

Note: `--restore-test` is the correct flag (not `--test-restore`). This is verified in `cli/internal/backup/verify.go` field `RestoreTest`.

Cross-link: [DR Runbook](operations/dr-runbook.md) uses `backup verify --restore-test` as the weekly automated integrity check.

### `backup prune`

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | false | Preview only |
| `--keep-daily` | `7` | Keep last N daily backups |
| `--keep-weekly` | `4` | Keep last N weekly backups |
| `--keep-monthly` | `12` | Keep last N monthly backups |
| `--format` | `""` | Output format: `json` |

### `backup config`

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `""` | Output format: `json` |
| `--install-cron` | false | Install systemd timers for backup, WAL, prune, verify |
| `--full-at` | `03:00` | Full backup time UTC (HH:MM) when `--install-cron` |
| `--wal-every` | `15m` | WAL checkpoint interval when `--install-cron` |
| `--prune-at` | `04:00` | Prune time UTC (HH:MM) when `--install-cron` |
| `--verify-on` | `Sun` | Weekly restore-test day when `--install-cron` |
| `--verify-at` | `05:00` | Weekly restore-test time UTC when `--install-cron` |
| `--remote` | `""` | Override configured remote when `--install-cron` |
| `--unit-dir` | `/etc/systemd/system` | Systemd unit directory when `--install-cron` |
| `--dry-run` | false | Print unit files without writing when `--install-cron` |

### `backup status`

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `""` | Output format: `json` |

## Examples

```bash
# Generate the age keypair before first encrypted backup
nself backup init-key

# Create a full encrypted backup tagged for traceability
nself backup create --type full --encrypt --tag pre-deploy

# List backups newer than a week, JSON for piping
nself backup list --since 7d --format json

# Restore latest backup, restoring only Postgres data
nself backup restore latest --only pg --yes

# Verify the latest backup with a real restore-test container
nself backup verify latest --restore-test

# Install systemd timers so backups run automatically
sudo nself backup config --install-cron --full-at 02:30
```

## Archive Format

`nself backup` creates archives that follow the `nself-backup-v1` spec. The spec covers the archive filename pattern, `manifest.json` structure, `checksums.sha256` format, encryption contract, and the exact extraction order used by `restore`. See `.claude/docs/operations/BACKUP-FORMAT.md`.

## See Also

- [[cmd-dr]], disaster recovery operations
- [[cmd-secrets]], manage encryption keys
- [[cmd-db]], database operations
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
