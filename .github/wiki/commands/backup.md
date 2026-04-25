# backup

Backup, restore, verify, and schedule nSelf project data.

## Synopsis

```bash
nself backup <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
|---|---|
| `create` | Create a local backup (full, wal, metadata, minio, all) |
| `stream` | Stream an encrypted backup to a remote destination |
| `restore` | Restore from a local backup file |
| `restore-remote` | Stream and restore directly from a remote URL |
| `resume` | Resume an interrupted streaming backup |
| `schedule` | Schedule recurring streaming backups via systemd timers |
| `verify` | Verify backup integrity |
| `list` | List available backups |
| `prune` | Remove old backups by retention policy |
| `config` | View backup configuration and install systemd timers |
| `status` | Show backup subsystem status |
| `init-key` | Generate age encryption keypair |

---

## backup stream

Stream a live backup to S3, R2, Backblaze B2, GCS, or Azure Blob. No temp files written.

### Pipeline

```
pg_dump (streaming) | age (encrypt) | rclone rcat (multipart upload)
```

### Usage

```bash
nself backup stream --to <url> [--recipient <key>] [--dry-run]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--to` | `NSELF_BACKUP_DESTINATION` | Destination URL (rclone remote path) |
| `--recipient` | `NSELF_BACKUP_RECIPIENT` | Encryption recipient: age key, SSH key, or `github:<user>` (repeatable) |
| `--dry-run` | false | Preview without running |

### Examples

```bash
# Stream encrypted backup to S3
nself backup stream --to s3:mybucket/backups --recipient age1abc123

# Use GitHub SSH keys for encryption
nself backup stream --to r2:mybucket/backups --recipient github:myusername

# Use env-configured destination (no flags needed)
nself backup stream

# Dry run to confirm destination
nself backup stream --to b2:mybucket --dry-run
```

### Requirements

- `pg_dump` on PATH
- `rclone` on PATH (configured with target remote)
- `age` on PATH (only required when encryption recipients are specified)

### Environment variables

| Variable | Description |
|---|---|
| `NSELF_BACKUP_DESTINATION` | Default destination URL |
| `NSELF_BACKUP_RECIPIENT` | Default age/SSH public key (space-separated for multiple) |
| `NSELF_BACKUP_CHUNK_MB` | Multipart chunk size in MB (default: 64, handled by rclone) |
| `AWS_ACCESS_KEY_ID` | S3/R2/B2 access key |
| `AWS_SECRET_ACCESS_KEY` | S3/R2/B2 secret key |

---

## backup restore-remote

Restore a backup directly from a remote URL. No local disk space required.

### Pipeline

```
rclone cat <from> | age --decrypt | pg_restore
```

### Usage

```bash
nself backup restore-remote --from <url> [--key <identity-file>] [--yes]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--from` | — | Source URL (rclone remote path) |
| `--key` | `~/.config/nself/age-key.txt` | Path to age identity file |
| `--yes` | false | Skip confirmation on production |

### Examples

```bash
# Restore encrypted backup from S3
nself backup restore-remote --from s3:mybucket/backups/myproject_stream_20260423.sql.age \
  --key ~/.config/nself/age-key.txt

# Restore unencrypted backup
nself backup restore-remote --from r2:mybucket/backups/myproject_stream_20260423.sql
```

---

## backup resume

Resume a previously interrupted streaming backup.

Since rclone `rcat` uploads are not resumable at the protocol level, resume re-streams the full backup and overwrites the partial remote object at the same key.

### Usage

```bash
nself backup resume <backup-id>
```

Resume state is stored in `~/.nself/backup-state/<id>.json`.

---

## backup schedule

Install a systemd timer to run `nself backup stream` on a cron schedule.

### Usage

```bash
nself backup schedule --cron "0 2 * * *" --to <url> [--recipient <key>] [--dry-run]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--cron` | — | Cron expression (e.g. `0 2 * * *`) |
| `--to` | `NSELF_BACKUP_DESTINATION` | Destination URL |
| `--recipient` | — | Default encryption recipient |
| `--unit-dir` | `/etc/systemd/system` | Systemd unit directory |
| `--dry-run` | false | Print unit files without writing |

### Examples

```bash
# Schedule nightly encrypted backup at 02:00 UTC
nself backup schedule --cron "0 2 * * *" --to s3:mybucket/backups --recipient age1abc123

# Preview the systemd units
nself backup schedule --cron "0 2 * * *" --to r2:mybucket --dry-run
```

### Status

After scheduling, check the timer with:

```bash
nself backup status
systemctl status nself-backup-stream.timer
```

---

## backup create

Create a local backup (written to `BACKUP_DIR`, default `./backups`).

```bash
nself backup create [--type full|wal|metadata|minio|all] [--encrypt] [--tag <label>] [--dry-run]
```

---

## backup restore

Restore from a local backup file.

```bash
nself backup restore <backup-id|latest> [--only pg,minio,metadata] [--decrypt-key <file>] [--yes]
```

---

## backup verify

Verify backup integrity, optionally running a restore test in a temporary container.

```bash
nself backup verify <backup-id|latest> [--restore-test] [--cleanup] [--keep]
```

---

## backup list

List backups in the local backup directory.

```bash
nself backup list [--remote <name>] [--since 24h] [--format table|json]
```

---

## backup prune

Remove backups beyond the retention policy.

```bash
nself backup prune [--keep-daily 7] [--keep-weekly 4] [--keep-monthly 12] [--dry-run]
```

---

## backup status

Show backup subsystem status: last run, next scheduled run, retention policy.

```bash
nself backup status [--format json]
```

---

## backup init-key

Generate an age encryption keypair for backup encryption.

```bash
nself backup init-key
```

Outputs the public key to add to `.env` as `BACKUP_AGE_RECIPIENTS`.

---

## Related

- [[cmd-pitr]] — point-in-time recovery via WAL archiving
- [[Home]]
