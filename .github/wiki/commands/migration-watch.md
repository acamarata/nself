# `nself migration watch`

Watch TypeScript, Go, and Dart model files and propose SQL migrations automatically on change.

## Summary

`nself migration watch` runs a long-lived process that monitors model files for schema changes.
When a field is added to a struct, interface, or class, the AI plugin proposes a candidate
`ALTER TABLE ADD COLUMN` migration and stages it in `migrations/`.

The additive-only guard ensures no `DROP COLUMN`, `DROP TABLE`, or `TRUNCATE` is written.
`NSELF_MIGRATION_AUTO_APPLY` is always false. Run `nself db migrate up` to apply a staged file.

## Usage

```bash
nself migration watch [flags]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--paths` | `./src,./lib,./internal` | Comma-separated directories to watch |
| `--extensions` | `.ts,.go,.dart` | File extensions to monitor |
| `--debounce` | `500` | Debounce delay in milliseconds |
| `--ai-endpoint` | `$NSELF_AI_ENDPOINT` | AI plugin base URL |
| `--migrations-dir` | `migrations` | Directory for staged migration files |

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `NSELF_MIGRATION_WATCH_PATHS` | `./src,./lib,./internal` | Directories to watch |
| `NSELF_MIGRATION_WATCH_EXTENSIONS` | `.ts,.go,.dart` | File extensions |
| `NSELF_MIGRATION_WATCH_DEBOUNCE_MS` | `500` | Debounce in milliseconds |
| `NSELF_MIGRATION_AUTO_APPLY` | `false` | Always false, never overridden in production |
| `NSELF_MIGRATION_AI_CONFIDENCE_MIN` | `0.75` | Minimum AI confidence threshold |

## Examples

```bash
nself migration watch
nself migration watch --paths ./src,./internal --debounce 1000
```

## Output

```
 nSelf Migration Watcher
  Watching for model changes in /your/project/src
  Debounce:   500 ms
  AI:         http://localhost:3001

Schema change detected in src/models/user.ts:
+ User.deletedAt (string)

Migration staged: migrations/20260423152301_auto_schema_change_in_user.sql (confidence: 92%)
Review the file, then run: nself db migrate up
```

Press `Ctrl+C` to stop the watcher.

---

# `nself migrate generate`

Generate a SQL migration from a natural-language description using the nSelf AI plugin.

## Summary

`nself migrate generate` calls the nself-ai plugin to propose a SQL migration file from a
plain-English description. The AI reads the prompt, produces candidate SQL, and stages a
migration file in `migrations/`. You review the SQL and confirm or reject the proposal.

The additive-only guard ensures no `DROP`, `TRUNCATE`, or `DELETE` statement is ever written
to disk. Blocked statements appear as comments in the migration file for manual review.

## Usage

```bash
nself migrate generate "<description>"
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--yes`, `-y` | `false` | Accept the proposal without interactive confirmation |
| `--dry-run` | `false` | Preview the generated SQL without writing any files |
| `--confidence` | `0.75` | Minimum AI confidence to accept the proposal (0.0-1.0) |
| `--migrations-dir` | `migrations` | Directory to write the migration file into |
| `--ai-endpoint` | `$NSELF_AI_ENDPOINT` or `http://localhost:3001` | nSelf AI plugin endpoint |

## Examples

```bash
# Basic usage — interactive confirm
nself migrate generate "add soft-delete column to users"

# Auto-accept without prompt
nself migrate generate "add index on orders.created_at" --yes

# Preview only, no files written
nself migrate generate "add tenant_id to all np_ tables" --dry-run

# Custom confidence threshold
nself migrate generate "rename user_name to display_name" --confidence 0.85
```

## Output

```
 AI Migration Generator
  Prompt:   add soft-delete column to users
  Endpoint: http://localhost:3001
  Min conf: 0.75

Proposed SQL (confidence: 92%, model: claude-3-5-sonnet):

  ------------------------------------------------------------
  ALTER TABLE np_users ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
  CREATE INDEX CONCURRENTLY idx_np_users_deleted_at ON np_users (deleted_at)
    WHERE deleted_at IS NOT NULL;
  ------------------------------------------------------------

Accept this proposal and stage the migration file? [y/N] y
  Migration staged: migrations/20260423143201_auto_add_soft_delete_column_to_users.sql
  Review the file, then run: nself db migrate up
```

## Additive-Only Safety Guard

The guard blocks the following SQL statement types regardless of what the AI proposes:

- `DROP TABLE`
- `DROP COLUMN` (via `ALTER TABLE ... DROP ...`)
- `TRUNCATE`
- `DELETE`

Blocked statements are preserved as comments in the migration file with a `WARNING` header so
you can review them and decide whether to apply them manually.

`ALTER TABLE ... ADD COLUMN` is always safe and passes through the guard.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `NSELF_AI_ENDPOINT` | AI plugin base URL. Overrides the default `http://localhost:3001`. |
| `NSELF_MIGRATION_AUTO_APPLY` | Must not be `true` in production profiles. The CLI ignores this for `prod`/`production` profiles. |
| `NSELF_MIGRATION_AI_CONFIDENCE_MIN` | Sets the default confidence floor when no `--confidence` flag is provided. |

## Integration with `nself db migrate`

The staged migration file is a standard SQL file compatible with `nself db migrate up`:

```bash
nself migrate generate "add last_login_at to users" --yes
nself db migrate up
```

## See Also

- [[cmd-db]] — `nself db migrate` — apply, revert, and list migrations
- [[cmd-migrate]] — `nself migrate` — v1 to v2 project migration

[[Home]]
