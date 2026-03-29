# nself migrate

> Detect and migrate nSelf v0.x artifacts to v1.0.0.

## Synopsis

```
nself migrate [subcommand] [flags]
```

## Description

`nself migrate` helps you upgrade an existing nSelf v1 project to v2. Running `nself migrate` without a subcommand performs a detection scan — the same as `nself migrate detect` — and reports which v1 artifacts are present and what needs to change.

The `run` subcommand performs the full automated migration: it stops running containers, backs up the current project state to `.nself/backup/{timestamp}/`, moves nginx configs to the v2 directory layout, regenerates `docker-compose.yml`, nginx, and SSL certificates, and prints a summary of every change made. The migration is idempotent — running it on an already-v2 project exits cleanly with no changes.

If anything goes wrong, use `nself migrate rollback` to restore from the most recent backup.

> **Note:** `nself migrate` manages v1→v2 project migration. For database schema migrations, see [[cmd-db]].

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `detect` | Detect v1 artifacts in the current project (alias for default scan) |
| `run` | Full v1→v2 migration: stop containers, backup, move configs, regenerate |
| `rollback` | Restore from the most recent migration backup |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--backup` | `""` | Specific backup timestamp to restore (for `rollback`, e.g. `20260328-143022`) |
| `--list` | false | List available backups with timestamps and sizes (for `rollback`) |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Scan for v1 artifacts
nself migrate
nself migrate detect

# Perform the v1→v2 migration
nself migrate run

# Restore from the most recent backup
nself migrate rollback

# List available backups
nself migrate rollback --list

# Restore from a specific backup
nself migrate rollback --backup 20260328-143022
```

← [[Commands]] | [[Home]] →
