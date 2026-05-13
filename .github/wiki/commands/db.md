# db

Database operations: migrations, backups, restore, seed, and shell.

## Synopsis

```bash
nself db <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
|---|---|
| `migrate` | Manage database migrations (up/down/status/create/apply) |
| `seed` | Run seed data |
| `backup` | Create pg_dump backup |
| `restore` | Restore from backup |
| `shell` | Open psql interactive shell |
| `drop` | Drop the project database (DESTRUCTIVE) |
| `reset` | Drop and recreate database (DESTRUCTIVE) |
| `lint` | Audit database schema and RLS policies |
| `hasura` | Hasura metadata operations |

---

## db migrate

```bash
nself db migrate <subcommand> [flags]
```

### Flags (persistent)

| Flag | Description |
|---|---|
| `--plugin <name>` | Scope migration operations to a specific plugin |

### db migrate up

Apply all pending migrations in lexicographic order.

```bash
nself db migrate up [--dry-run] [--migration-dir <path>] [--plugin <name>]
```

| Flag | Description |
|---|---|
| `--dry-run` | List pending migrations without applying them |
| `--migration-dir <path>` | Apply all `.sql` files in `<path>` in lexicographic order, skipping already-applied files |

**Examples:**

```bash
# Apply all pending migrations
nself db migrate up

# Preview pending migrations
nself db migrate up --dry-run

# Apply external directory of SQL files (G-008)
nself db migrate up --migration-dir /path/to/plugin/migrations
```

### db migrate apply

Apply a single SQL migration file by path (G-008).

```bash
nself db migrate apply --file <path>
```

| Flag | Required | Description |
|---|---|---|
| `--file <path>` | yes | Path to the `.sql` migration file to apply |

**Double-apply protection:** if the file's filename is already recorded in `schema_versions`, the command prints `already applied: <filename> (skipped)` and exits cleanly with code 0. It does not re-execute the SQL.

**Checksum tracking:** the SHA-256 checksum of the file is stored in `nself_ops.migrations` for audit purposes.

**Non-transactional detection:** SQL files containing `CREATE INDEX CONCURRENTLY`, `DROP INDEX CONCURRENTLY`, `REINDEX CONCURRENTLY`, or `ALTER TYPE` are run outside a transaction automatically.

**Examples:**

```bash
# Apply a specific external RLS migration
nself db migrate apply --file /path/to/plugin/rls_policy.sql

# Apply a migration from a plugin's migration directory
nself db migrate apply --file ~/.nself/plugins/claw/migrations/20240115_rls.sql
```

**Use case:** plugin-claw external RLS migrations can be applied via CLI without requiring `nself db shell` as a workaround.

### db migrate down

Revert the most recently applied migration.

```bash
nself db migrate down
```

Looks for a corresponding `.down.sql` file next to the original migration.

### db migrate status

Show applied and pending migrations.

```bash
nself db migrate status
```

Reports: migration name, status (applied/pending), and timestamp for applied migrations.

### db migrate create

Create a new migration file pair (up + down).

```bash
nself db migrate create <name>
```

Creates `migrations/<timestamp>_<name>.sql` and `migrations/<timestamp>_<name>.down.sql`.

Name must be lowercase alphanumeric with underscores or hyphens only.

---

## db seed

Run seed data for the current environment.

```bash
nself db seed [file]
nself db seed run [--env <env>] [--fixture <name>] [--reset]
nself db seed list
nself db seed verify [--fixture <name>]
nself db seed graph
```

---

## db backup

```bash
nself db backup [file]
nself db backup list [--format table|json]
```

---

## db restore

```bash
nself db restore <file> [--overwrite] [--yes]
```

---

## db shell

Open an interactive psql shell inside the project's Postgres container.

```bash
nself db shell
```

---

## db drop

Drop the project database. This is destructive and irreversible.

```bash
nself db drop [--yes]
```

---

## db reset

Drop and recreate the project database from scratch.

```bash
nself db reset [--force] [--yes]
```

---

## db lint

Audit database schema and RLS policies.

```bash
nself db lint [--format table|json] [--rls] [--metric] [--matrix] [--remediate]
```

---

## db hasura

Hasura metadata operations.

```bash
nself db hasura console
nself db hasura metadata apply
nself db hasura metadata export
nself db hasura metadata reload
nself db hasura diff
nself db hasura validate
```

---

## See also

- [[Architecture]] — stack overview including migration engine
- [[cmd-plugin-dev]] — plugin development, including plugin-specific migrations
- [[backup]] — full backup subcommand reference

[[Home]]
