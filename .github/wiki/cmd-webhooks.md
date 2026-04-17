# nself webhooks

> Manage webhook processing and outbox.

## Synopsis

```
nself webhooks <subcommand> [flags]
```

## Description

`nself webhooks` manages the durable outbox queue used by the webhook subsystem. When a webhook event cannot be written to Postgres at receipt time (transient DB outage, network blip), the event is appended to a durable on-disk outbox at `/var/lib/nself/webhook-outbox/` (or the path in `NSELF_WEBHOOK_OUTBOX_DIR`) so it can be retried later.

`webhooks outbox status` reports the outbox directory, current depth (number of queued events), and the file names of pending events. Output can be table (default) or JSON for machine consumers.

This command is read-only today; processing of queued events is performed by the webhook worker on a schedule, not by the CLI.

## Subcommands

| Name | Description |
|------|-------------|
| `outbox status` | Show current outbox status |

## Flags

### `webhooks outbox status`

| Flag | Default | Description |
|------|---------|-------------|
| `--format`, `-f` | `text` | Output format: `text` or `json` |

## Examples

```bash
# Show outbox depth and queued event filenames
nself webhooks outbox status

# JSON form for a metrics pipeline
nself webhooks outbox status --format json
```

## See Also

- [[cmd-queue]] — async job queues
- [[cmd-watchdog]] — self-healing watchdog
- [[cmd-status]] — service health
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
