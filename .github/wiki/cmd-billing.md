# nself billing

<!-- BEGIN PROSE:summary -->
> **SHIPPED (v1.1.1):** `billing usage`, `billing invoice-preview`, `billing report`, and `billing retry-event` are all live. Stripe is the backing integration for `invoice-preview` and `retry-event`.
<!-- END PROSE:summary -->

## Synopsis

```
nself billing <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself billing` exposes per-tenant billing and usage metering for multi-tenant ɳSelf deployments. It reports usage metrics, generates billing reports across tenants, and retries failed Stripe outbox events.

`billing usage` queries the metering store for a single tenant and a month, optionally as CSV or JSON. `billing report` rolls usage up across tenants for a billing period and is suitable for accounting export. `billing retry-event` re-enqueues a failed Stripe outbox event by ID, useful when transient API errors leave events stuck.

`billing invoice-preview` is reserved: the underlying Stripe integration is pending. The command currently returns a clear error pointing operators to `billing usage` and the Stripe dashboard.

> Billing operations: usage, invoice-preview, report, retry-event.

### `billing usage <tenant-id>`
### `billing report`
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `invoice-preview` | Preview next Stripe invoice (requires STRIPE_SECRET_KEY) |
| `report` | Generate billing report across tenants |
| `retry-event` | Re-enqueue a failed Stripe outbox event |
| `usage` | Show usage metrics for a tenant |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
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
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-tenant]], tenant lifecycle management
- [[cmd-license]], license keys
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
