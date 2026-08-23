# nself dr

<!-- BEGIN PROSE:summary -->
> Disaster recovery operations: drill, promote-standby, reconfigure-dns, rollback, fence.
<!-- END PROSE:summary -->

## Synopsis

```
nself dr <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself dr` runs disaster recovery procedures. It covers planned drills, promoting a warm standby to primary, rewriting DNS records to a new primary IP, rolling back a promotion, and fencing the cluster (Redis read-only flag) to prevent split-brain writes during failover.

`dr drill` is the primary verification command (v1.1.1). Supported scenarios: `cold-start`, `region-failover`, `data-corruption`. Cold-start provisions a Hetzner VM via hcloud, restores the latest backup via ssh, runs the full smoke-query catalog, records RTO, and writes a dated report to `~/.claude/backups/nself-staging/dr/`. `--install-cron` installs the monthly drill systemd timer (`nself-dr-drill.timer`).

**Scenario support (v1.1.1):**
- `cold-start`, fully implemented; provisions VM, restores, verifies, records RTO
- `region-failover`, coordinates with `nself region` (`list`, `add`, `promote`) for multi-region failover drills
- `data-corruption`, exercises `nself pitr restore` (pgbackrest WAL replay) on a clone

`dr promote-standby` requires production confirmation unless `--yes` is passed. `dr reconfigure-dns --ip <new-ip>` updates DNS to point traffic at the new primary. `dr rollback` demotes the promoted standby and resyncs from the original primary. `dr fence` sets a `read_only=true` flag in Redis that the application layer must honor.

### `dr drill`
### `dr promote-standby`
### `dr reconfigure-dns`
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
| `drill` | Execute a DR drill |
| `fence` | Set read-only flag in Redis for split-brain prevention |
| `promote-standby` | Promote warm standby to primary |
| `reconfigure-dns` | Update DNS records to point to new primary |
| `rollback` | Demote promoted standby and resync from original primary |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Dry-run a cold-start drill
nself dr drill --dry-run

# Run an actual cold-start drill now
sudo nself dr drill --now

# Install the monthly drill timer on the ops host
sudo nself dr drill --install-cron --hetzner-project camarata

# Print the cloud-init template that would be used
nself dr drill --render-cloud-init

# Promote the warm standby to primary in eu-west
nself dr promote-standby --region eu-west --yes

# Point DNS at the new primary IP
nself dr reconfigure-dns --ip 5.75.235.42

# Fence writes during a manual failover
nself dr fence
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-backup]], backup operations
- [[cmd-promote]], environment promotion
- [[cmd-watchdog]], self-healing watchdog
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
