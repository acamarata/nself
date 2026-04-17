# nself promote

> Promote one environment to another (e.g. staging to prod).

## Synopsis

```
nself promote <source> <target> [flags]
nself promote rollback [flags]
```

## Description

`nself promote` moves configurations, migrations, and services from one environment to another. The most common path is `staging → prod`, but any source-to-target pair is allowed. Use `--dry-run` first to preview what would change without writing anything.

Production promotions are gated: a target of `prod` requires `--confirm prod` and an `--approve-id` (a ticket or change-management ID). Before applying, the command snapshots the current state into a backup tag so the operation is reversible. If the promotion fails, the message includes the rollback command with the snapshot tag.

`nself promote rollback` reverts to the most recent pre-promotion backup, or to a specific tag with `--tag`. Rollback is itself logged for audit.

## Subcommands

| Name | Description |
|------|-------------|
| `rollback` | Rollback to pre-promotion backup |

## Flags

### `promote <source> <target>`

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | false | Preview changes without applying |
| `--approve-id` | `""` | Approval ticket ID (required for non-dry-run) |
| `--confirm` | `""` | Confirmation target (required value `prod` for `prod` target) |
| `--json` | false | JSON output |

### `promote rollback`

| Flag | Default | Description |
|------|---------|-------------|
| `--tag` | `""` | Specific backup tag to restore (default: latest pre-promote) |

## Examples

```bash
# Preview a staging-to-prod promotion
nself promote staging prod --dry-run

# Apply the promotion with explicit approval
nself promote staging prod --approve-id TICKET-1234 --confirm prod

# Roll back the most recent promotion
nself promote rollback

# Roll back to a specific snapshot tag
nself promote rollback --tag pre-promote-2026-04-17T03:00Z
```

## See Also

- [[cmd-env]] — multi-environment management
- [[cmd-dr]] — disaster recovery
- [[cmd-backup]] — backup operations
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
