# nself region

<!-- BEGIN PROSE:summary -->
> Manage multi-region Postgres replicas for a self-hosted nSelf deployment.
<!-- END PROSE:summary -->

## Synopsis

```
nself region <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself region` configures and manages read replicas in additional geographic regions. Each region gets its own Postgres instance that replicates from the primary via WAL streaming. Traffic routing follows the strategy set during `nself init`.

All `region` subcommands require the `multi_region_enabled` feature flag to be on. Enable it once with `nself flag set multi_region_enabled true` before using any subcommand. Running a subcommand without the flag returns an error with the enable command printed.

The region command is **planned** and requires approval of the UD-12 minor release. It is present in the source tree but gated pending that release.

### nself region add
Register a new region by supplying a region identifier and a Postgres connection URL for the replica.
```
nself region add [flags]
```
### nself region list
List all configured regions and their current status. No flags.
### nself region status
Show replication lag and routing status for all regions. No flags.
### nself region promote
Promote a replica region to primary. nSelf checks WAL replication lag before proceeding and updates DNS records (60-second TTL) to point at the new primary.
```
nself region promote [flags]
```
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
| `add` | Add a replica region |
| `list` | List configured regions |
| `promote` | Promote a replica to primary (failover) |
| `status` | Show replication and routing status for all regions |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
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
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [cmd-flag.md](cmd-flag.md) — enable or disable feature flags such as multi_region_enabled
- [cmd-doctor.md](cmd-doctor.md) — verify stack health across all regions
- [cmd-build.md](cmd-build.md) — regenerate docker-compose after adding a region
- [Commands.md](Commands.md) — full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
