# nself db

> Database operations: migrations, backups, restore, seed, and shell.

## Synopsis

```
nself db <subcommand> [flags]
```

## Description

`nself db` provides a complete interface for interacting with the PostgreSQL database in your ɳSelf stack. It covers the full database lifecycle: migrations, seeding, backup and restore, an interactive shell, and Hasura metadata management.

All operations target the PostgreSQL container managed by ɳSelf. The `shell` subcommand opens an interactive `psql` session inside the container. The `backup` and `restore` subcommands use `pg_dump` and `pg_restore` and produce SQL dump files that are compatible with standard PostgreSQL tooling.

The `hasura` subgroup controls Hasura metadata, useful for applying tracked tables and permissions from version-controlled metadata files without opening the Hasura Console.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `migrate up` | Apply pending migrations |
| `migrate down` | Revert the last migration |
| `migrate status` | Show migration status |
| `migrate create <name>` | Create a new migration file |
| `seed [file]` | Run seed data (default seed file if not specified) |
| `backup [file]` | Create a `pg_dump` backup (timestamped filename if omitted) |
| `backup list` | List available backups with size and date |
| `restore <file>` | Restore database from a backup file |
| `shell` | Open interactive `psql` shell in the PostgreSQL container |
| `drop` | Drop the project database , **DESTRUCTIVE** |
| `reset` | Drop and recreate the database , **DESTRUCTIVE** |
| `hasura console` | Open the Hasura Console in browser |
| `hasura metadata apply` | Apply Hasura metadata from files |
| `hasura metadata export` | Export Hasura metadata to files |
| `hasura metadata reload` | Reload the metadata cache |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--plugin` | `""` | Migrate a specific plugin schema (for `migrate` subcommands) |
| `--force`, `-f` | false | Skip confirmation prompt (for `reset`) |
| `--yes` | false | Skip confirmation prompt (for `drop`, `reset`, `restore`) |
| `--overwrite` | false | Allow overwriting existing data (for `restore`) |
| `--format` | `""` | Output format: `table` (default) or `json` (for `backup list`) |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Apply all pending migrations
nself db migrate up

# Apply migrations for a specific plugin
nself db migrate up --plugin ai

# Revert the last migration
nself db migrate down

# Check migration status
nself db migrate status

# Create a new migration file
nself db migrate create add_users_table

# Run default seed data
nself db seed

# Run a specific seed file
nself db seed seeds/dev.sql

# Create a timestamped backup
nself db backup

# Backup to a specific file
nself db backup /tmp/backup-2026-03-28.sql

# Restore from a backup
nself db restore /tmp/backup-2026-03-28.sql

# Open an interactive psql shell
nself db shell

# Drop the project database (interactive confirmation)
nself db drop

# Drop without confirmation (CI)
nself db drop --yes

# Drop and recreate database (interactive confirmation)
nself db reset

# Drop and recreate without confirmation (CI)
nself db reset --force

# List available backups (table format)
nself db backup list

# List available backups in JSON format
nself db backup list --format json

# Open Hasura Console
nself db hasura console

# Apply Hasura metadata
nself db hasura metadata apply

# Export Hasura metadata
nself db hasura metadata export

# Reload metadata cache
nself db hasura metadata reload
```

← [[Commands]] | [[Home]] →
