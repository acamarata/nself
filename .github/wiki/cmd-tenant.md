# nself tenant

<!-- BEGIN PROSE:summary -->
> **PREVIEW (v1.1.1):** tenant CLI works at the data and RLS layer. Provisioning automation, Stripe billing charges, license revocation on destroy, and runtime suspension are still being hardened. Do NOT use for paying-customer onboarding in production deployments without manual verification of the full lifecycle.
<!-- END PROSE:summary -->

## Synopsis

```
nself tenant <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself tenant` manages tenant lifecycle in multi-tenant ɳSelf deployments. Each tenant has a slug, a plan (`basic`, `pro`, `elite`, `business`, `business-plus`, `enterprise`), and an audit trail.

`tenant create` provisions a new tenant with a starting plan. `tenant upgrade` changes the plan in place. `tenant suspend` halts a tenant's access while preserving data; a `--reason` is required so the audit trail explains why. `tenant destroy` is the hard-delete path and requires `--confirm-name <slug>` to proceed. `tenant audit` queries the per-tenant audit log, optionally filtered by `--since` time window and rendered as table or JSON.

Plans must match server-side validation. Pair this command with `nself billing` for usage and invoicing.

> Tenant management: create, upgrade, suspend, destroy, audit.

### `tenant create <slug>`
### `tenant upgrade <slug>`
### `tenant suspend <slug>`
### `tenant destroy <slug>`
### `tenant audit <tenant-id>`
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
| `audit` | Query tenant audit log |
| `create` | Create a new tenant |
| `destroy` | Hard-delete a tenant (DESTRUCTIVE) |
| `suspend` | Suspend a tenant |
| `upgrade` | Change a tenant's plan |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Create a new tenant on the basic plan
nself tenant create acme-corp

# Create on the pro plan from the start
nself tenant create acme-corp --plan pro

# Upgrade to elite
nself tenant upgrade acme-corp --plan elite

# Suspend with an explicit reason
nself tenant suspend acme-corp --reason "non-payment 30d overdue"

# Destroy a tenant after explicit confirmation
nself tenant destroy acme-corp --confirm-name acme-corp

# Audit the last week
nself tenant audit acme-corp --since 7d

# Export audit log as JSON
nself tenant audit acme-corp --since 30d --format json
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-billing]], usage and invoicing
- [[cmd-license]], license keys
- [[cmd-secrets]], environment-scoped secrets
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
