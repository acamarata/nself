# Database Migrations

> Internal reference for how ɳSelf tracks and applies database schema migrations.

## Overview

ɳSelf manages PostgreSQL schema migrations through two tracking tables:

| Table | Purpose |
|---|---|
| `np_common.schema_versions` | Lightweight applied-migration log (name + timestamp) |
| `nself_ops.migrations` | Audit log with migration ID, name, checksum, and applied timestamp |

Both tables are updated atomically on every migration apply and revert. If either update fails, the transaction rolls back and neither table is modified.

## Apply (`nself db migrate up`)

For each pending migration file:

- **Transactional DDL** (default): the migration SQL and both INSERT statements run inside a single `BEGIN … COMMIT` block. `SET LOCAL lock_timeout = '5s'` and `SET LOCAL statement_timeout = '60s'` are applied before the DDL to prevent blocking production deployments.
- **Non-transactional DDL** (`CREATE INDEX CONCURRENTLY`, `ALTER TYPE ADD VALUE`, etc.): the DDL runs outside a transaction (required by PostgreSQL). Immediately after, both INSERT statements are wrapped in their own `BEGIN … COMMIT` block so that the recording step is still atomic.

In both cases, a failure rolls back to a clean state — no orphan rows are left in either tracking table.

## Revert (`nself db migrate down`)

`MigrateDown` reads the most recently applied migration from `np_common.schema_versions`, locates the corresponding `.down.sql` file, and runs a single transaction:

```sql
BEGIN;
<down migration DDL>
DELETE FROM np_common.schema_versions WHERE name = '<name>';
DELETE FROM nself_ops.migrations WHERE name = '<name>';
COMMIT;
```

Both `DELETE` statements execute inside the same transaction as the down DDL. Either all three succeed together, or the transaction rolls back and both tracking tables remain unchanged.

## Double-apply protection

`ApplyFile` checks `np_common.schema_versions` before executing a migration. If the filename is already present:

1. The stored checksum in `nself_ops.migrations` is compared against the file's current SHA-256.
2. If the checksums match, the migration is skipped with `(skipped=true, nil)`.
3. If the checksums differ, the function returns an error — the file was modified after it was applied and manual intervention is required.

## Migration file naming

| Pattern | Meaning |
|---|---|
| `NNN_<description>.sql` | Up migration |
| `NNN_<description>.down.sql` | Corresponding down migration |

Down files are excluded from `scanMigrations` and are only loaded by `MigrateDown`.

## SQL injection safety

All migration names inserted into SQL statements use single-quote escaping (`strings.ReplaceAll(name, "'", "''")`) before interpolation. This applies to both `DELETE` statements in `MigrateDown` and both `INSERT` statements in the recording step.

## Related pages

- [[cmd-db]] — `nself db` command reference
- [[cmd-migrate]] — v0.9→v1 project migration (separate from schema migrations)
- [[Home]]
