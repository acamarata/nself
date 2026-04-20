# nself dlq

> Manage dead-letter queues for nSelf plugins.

## Synopsis

```
nself dlq <subcommand>
nself dlq replay <plugin> [flags]
```

## Description

`nself dlq` manages dead-letter queues (DLQ) for nSelf plugins. DLQs accumulate
rows that failed processing (e.g. mux gmail rows that could not be delivered).

Use `nself dlq replay` after fixing the upstream bug to re-enqueue failed rows
without manually manipulating the database.

## Subcommands

| Subcommand | Description |
|---|---|
| `replay` | Re-enqueue DLQ rows for a plugin back to the work queue |

---

## nself dlq replay

Re-enqueue dead-letter queue rows for the named plugin.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--max-rows <n>` | 100 | Maximum rows to replay (prevents accidental floods) |
| `--filter <field=value>` | — | Filter rows (repeatable: `--filter status=quarantined`) |
| `--dry-run` | false | Preview replay without executing |
| `--base-url <url>` | `$NSELF_API_URL` | nSelf API base URL |

### WARNING

**Only replay after the upstream bug is fixed.** Replaying before fixing the root cause
causes rows to re-DLQ immediately, creating a loop. The command prominently warns you
before executing.

### Behavior

- Queries the plugin's DLQ endpoint: `GET /api/plugins/<plugin>/dlq?max=<n>&<filters>`
- Re-enqueues each row via: `POST /api/plugins/<plugin>/dlq/<id>/replay`
- Reports success/failure **per row** — failures are not aggregated
- Exits non-zero if any row fails

### Examples

```bash
# Preview what would be replayed
nself dlq replay mux --dry-run

# Replay up to 100 quarantined rows for the mux plugin
nself dlq replay mux --filter status=quarantined

# Replay at most 10 rows
nself dlq replay mux --max-rows 10

# Replay with multiple filters
nself dlq replay mux --filter status=quarantined --max-rows 50
```

### Exit Codes

| Code | Meaning |
|---|---|
| 0 | All rows replayed successfully |
| 1 | One or more rows failed — see per-row output |

### See Also

- [Observability](https://docs.nself.org/operations/observability) — DLQ metrics (S41-T02)
- [mux plugin](https://docs.nself.org/plugins/mux) — mux DLQ schema
