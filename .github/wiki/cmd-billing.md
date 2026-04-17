# nself billing

> Billing operations: usage, invoice-preview, report, retry-event.

## Synopsis

```
nself billing <subcommand> [flags] [args]
```

## Description

`nself billing` exposes per-tenant billing and usage metering for multi-tenant nSelf deployments. It reports usage metrics, generates billing reports across tenants, and retries failed Stripe outbox events.

`billing usage` queries the metering store for a single tenant and a month, optionally as CSV or JSON. `billing report` rolls usage up across tenants for a billing period and is suitable for accounting export. `billing retry-event` re-enqueues a failed Stripe outbox event by ID, useful when transient API errors leave events stuck.

`billing invoice-preview` is reserved: the underlying Stripe integration is pending. The command currently returns a clear error pointing operators to `billing usage` and the Stripe dashboard.

## Subcommands

| Name | Description |
|------|-------------|
| `usage <tenant-id>` | Show usage metrics for a tenant |
| `invoice-preview <tenant-id>` | Preview next Stripe invoice (Stripe integration pending) |
| `report` | Generate billing report across tenants |
| `retry-event <id>` | Re-enqueue a failed Stripe outbox event |

## Flags

### `billing usage <tenant-id>`

| Flag | Default | Description |
|------|---------|-------------|
| `--month` | `""` | Filter by month (`YYYY-MM`) |
| `--format` | `csv` | Output format: `csv` or `json` |

### `billing report`

| Flag | Default | Description |
|------|---------|-------------|
| `--tenant` | `""` | Filter by tenant slug |
| `--month` | `""` | Filter by month (`YYYY-MM`) |
| `--format` | `table` | Output format: `table` or `json` |

## Examples

```bash
# Show usage for a tenant for the current month
nself billing usage acme-corp

# Pull JSON usage for last month for finance
nself billing usage acme-corp --month 2026-03 --format json

# Generate a tabular billing report across all tenants for the month
nself billing report --month 2026-03

# Retry a stuck Stripe outbox event by ID
nself billing retry-event evt_1JxYXz2eZvKYlo2C
```

## See Also

- [[cmd-tenant]] — tenant lifecycle management
- [[cmd-license]] — license keys
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
