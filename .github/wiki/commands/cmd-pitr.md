# nself pitr

Point-in-time recovery (PITR) via continuous WAL archiving.

PITR archives PostgreSQL write-ahead log segments so that any second within the
retention window can be used as a restore target. A scheduled base backup
(`pg_basebackup`) combined with a stream of WAL segments lets you recover to
any point in the past, not just the last full backup.

---

## Synopsis

```bash
nself pitr enable       --to <destination> [flags]
nself pitr disable
nself pitr status
nself pitr base-backup  [--encrypt-recipient <age-pubkey>]
nself pitr restore      --to <timestamp-or-relative>
```

---

## Subcommands

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

| Flag | Default | Description |
|------|---------|-------------|
| `--to` | (required) | Destination URL for WAL archives (s3://, r2://, gcs://, etc.) |
| `--encrypt-recipient` | — | age public key to encrypt WAL segments |
| `--retention-days` | 7 | Number of days to retain WAL segments |
| `--wal-timeout` | 60 | `archive_timeout` in seconds , flush WAL segment at most this often |

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

| Flag | Default | Description |
|------|---------|-------------|
| `--encrypt-recipient` | — | age public key to encrypt the backup archive |

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

| Flag | Default | Description |
|------|---------|-------------|
| `--to` | (required) | Target time in RFC3339 or relative form |
| `--identity` | — | Path to age identity file for decrypting encrypted archives |
| `--auto-promote` | true | Automatically promote Postgres after recovery |

**Supported time formats:**

| Format | Example |
|--------|---------|
| RFC3339 | `2026-04-22T15:30:00Z` |
| Relative minutes | `30 minutes ago` |
| Relative hours | `2 hours ago` |
| Relative days | `1 day ago` |

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

## See also

- [[cmd-backup]], full and metadata backups
- [[cmd-restore]], restore from a pg_dump backup
- [Point-in-Time Recovery Guide](https://docs.nself.org/guides/point-in-time-recovery)

[[Home]]
