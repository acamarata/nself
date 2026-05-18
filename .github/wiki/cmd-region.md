# nself region

> Manage multi-region Postgres replicas for a self-hosted nSelf deployment.

## Synopsis

```
nself region <subcommand> [flags]
```

## Description

`nself region` configures and manages read replicas in additional geographic regions. Each region gets its own Postgres instance that replicates from the primary via WAL streaming. Traffic routing follows the strategy set during `nself init`.

All `region` subcommands require the `multi_region_enabled` feature flag to be on. Enable it once with `nself flag set multi_region_enabled true` before using any subcommand. Running a subcommand without the flag returns an error with the enable command printed.

The region command is **planned** and requires approval of the UD-12 minor release. It is present in the source tree but gated pending that release.

## Subcommands

| Name | Description |
|------|-------------|
| `add` | Register a new region and its Postgres replica |
| `list` | List all configured regions and their sync status |
| `status` | Show replication lag and health for all regions |
| `promote` | Promote a replica to become the new primary |

### nself region add

Register a new region by supplying a region identifier and a Postgres connection URL for the replica.

```
nself region add [flags]
```

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--region` | | string | | Region identifier (e.g. hel1) |
| `--pg-url` | | string | | Postgres URL for the replica |
| `--redis-url` | | string | `""` | Redis URL for the region (optional) |

### nself region list

List all configured regions and their current status. No flags.

### nself region status

Show replication lag and routing status for all regions. No flags.

### nself region promote

Promote a replica region to primary. nSelf checks WAL replication lag before proceeding and updates DNS records (60-second TTL) to point at the new primary.

```
nself region promote [flags]
```

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--region` | | string | | Region ID to promote to primary |

## Examples

```bash
# Enable the feature flag before first use
nself flag set multi_region_enabled true
```

```bash
# Add a replica in Hetzner Falkenstein with its Postgres URL
nself region add --region hel1 --pg-url postgres://user:pass@10.0.0.2:5432/mydb
```

```bash
# Add a replica with a dedicated Redis instance
nself region add --region ash1 --pg-url postgres://user:pass@10.0.1.2:5432/mydb --redis-url redis://10.0.1.3:6379
```

```bash
# List all configured regions and replication status
nself region list
```

```bash
# Check replication lag and routing for all regions
nself region status
```

```bash
# Promote hel1 to primary (DNS TTL: 60s)
nself region promote --region hel1
```

## See Also

- [cmd-flag.md](cmd-flag.md) — enable or disable feature flags such as multi_region_enabled
- [cmd-doctor.md](cmd-doctor.md) — verify stack health across all regions
- [cmd-build.md](cmd-build.md) — regenerate docker-compose after adding a region
- [Commands.md](Commands.md) — full command index

← [[Commands]] | [[Home]] →
