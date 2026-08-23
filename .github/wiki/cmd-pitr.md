# nself pitr

<!-- BEGIN PROSE:summary -->
> Point-in-time recovery: enable, disable, status, base-backup, restore.
<!-- END PROSE:summary -->

## Synopsis

```
nself pitr <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
Point-in-time recovery (PITR) via continuous WAL archiving.

PITR archives PostgreSQL write-ahead log segments so that any second within the
retention window can be used as a restore target. A scheduled base backup
(`pg_basebackup`) combined with a stream of WAL segments lets you recover to
any point in the past, not just the last full backup.

---

### pitr enable
Configure PostgreSQL for continuous WAL archiving and write the PITR config
snippet.
```bash
nself pitr enable --to s3://my-bucket/pitr
nself pitr enable --to s3://my-bucket/pitr --encrypt-recipient age1abc... --retention-days 14
```
Writes `.nself/pitr/postgresql.conf.d/pitr.conf`. Mount this file into the
PostgreSQL container to activate WAL archiving. Sets `wal_level = logical`
(satisfies both PITR and CDC logical replication, no separate setting needed).
**Flags:**
---
### pitr disable
Remove the PITR config snippet. WAL archiving stops on the next container restart.
```bash
nself pitr disable
```
---
### pitr status
Show the current retention window, WAL segment count, and the latest archived WAL timestamp.
```bash
nself pitr status
```
**Output:**
```
Base backups:    3
WAL segments:    1440
Oldest restore:  2026-04-15T02:00:00Z
Latest WAL:      2026-04-22T15:28:34Z
```
---
### pitr base-backup
Take a manual base backup right now using `pg_basebackup`. Useful before
maintenance windows or major schema changes.
```bash
nself pitr base-backup
nself pitr base-backup --encrypt-recipient age1abc...
```
The backup is recorded in `~/.nself/wal-catalog.json`.
**Flags:**
---
### pitr restore
Restore the PostgreSQL database to a specific point in time.
```bash
nself pitr restore --to 2026-04-22T15:30:00Z
nself pitr restore --to "30 minutes ago"
nself pitr restore --to "2 hours ago"
```
Steps performed:
1. Locate the latest base backup before the target time in `~/.nself/wal-catalog.json`.
2. Fetch and (if encrypted) decrypt the base backup.
3. Write `recovery_target_time` configuration to `.nself/pitr/recovery.conf`.
4. Stop the PostgreSQL container, apply recovery config, restart.
5. Poll until `pg_is_in_recovery()` returns false (up to 5 minutes).
**Flags:**
**Supported time formats:**
---
## Configuration

PITR behaviour is controlled by environment variables in `.env` / `.env.prod`:

| Variable | Default | Description |
|----------|---------|-------------|
| `NSELF_PITR_ENABLED` | `false` | Enable WAL archiving |
| `NSELF_PITR_DESTINATION` | — | Destination URL (uses B44 drivers) |
| `NSELF_PITR_RETENTION_DAYS` | `7` | WAL retention window in days |
| `NSELF_PITR_BASE_BACKUP_SCHEDULE` | `0 2 * * *` | Cron for daily base backup |
| `NSELF_PITR_ENCRYPT_RECIPIENT` | — | age public key for WAL encryption |
| `NSELF_PITR_WAL_ARCHIVE_TIMEOUT` | `60` | `archive_timeout` in seconds |

---

## WAL Catalog

`~/.nself/wal-catalog.json` tracks base backup timestamps and WAL segment
metadata. The restore command uses this catalog to find the right base backup
and segments for a given target time. It is updated automatically by
`nself pitr base-backup` and by the WAL archive command.

---

## Encryption

When `--encrypt-recipient` is specified, WAL segments and base backups are
encrypted with [age](https://github.com/FiloSottile/age) before being written
to the destination. To restore, provide the corresponding identity file:

```bash
nself pitr restore --to "30 minutes ago" --identity ~/.config/nself/myproject-age.key
```

Generate a keypair with `nself backup init-key`.

---

## Relation to nself db pitr

`nself db pitr` is a lower-level command that inspects the live Postgres
`archive_mode` and `pg_stat_archiver` settings. `nself pitr` is the
higher-level PITR workflow: it manages the WAL catalog, base backups, retention,
and full restore orchestration.

---
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `base-backup` | Take a manual pg_basebackup snapshot now |
| `disable` | Stop WAL archiving (removes PITR config snippet) |
| `enable` | Configure WAL archiving and enable PITR |
| `restore` | Restore database to a specific point in time |
| `status` | Show PITR retention window, segment count, and latest WAL timestamp |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
<!-- TODO(docs): needs human prose -->

```bash
nself pitr
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-backup]], full and metadata backups
- [[cmd-backup]], restore from a pg_dump backup
- [Point-in-Time Recovery Guide](https://nself.org/docs/guides/point-in-time-recovery)

[[Home]]
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
