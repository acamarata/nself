# nself dr

> Disaster recovery operations: drill, promote-standby, reconfigure-dns, rollback, fence.

## Synopsis

```
nself dr <subcommand> [flags]
```

## Description

`nself dr` runs disaster recovery procedures. It covers planned drills (cold-start a fresh VM from latest backup, region failover, data corruption rehearsal), promoting a warm standby to primary, rewriting DNS records to a new primary IP, rolling back a promotion, and fencing the cluster (Redis read-only flag) to prevent split-brain writes during failover.

`dr drill` is the primary verification command. With `--scenario cold-start`, it provisions a Hetzner VM via cloud-init, restores the latest backup, runs a smoke test, and tears down. `--install-cron` installs the monthly drill systemd timer (`nself-dr-drill.timer`). `--render-cloud-init` and `--render-alerts` print the templates the drill would use without executing.

`dr promote-standby` requires production confirmation unless `--yes` is passed. `dr reconfigure-dns --ip <new-ip>` updates DNS to point traffic at the new primary. `dr rollback` demotes the promoted standby and resyncs from the original primary. `dr fence` sets a `read_only=true` flag in Redis that the application layer must honor.

## Subcommands

| Name | Description |
|------|-------------|
| `drill` | Execute a DR drill |
| `promote-standby` | Promote warm standby to primary |
| `reconfigure-dns` | Update DNS records to point to new primary |
| `rollback` | Demote promoted standby and resync from original primary |
| `fence` | Set read-only flag in Redis for split-brain prevention |

## Flags

### `dr drill`

| Flag | Default | Description |
|------|---------|-------------|
| `--scenario` | `cold-start` | Drill scenario: `cold-start`, `region-failover`, `data-corruption` |
| `--dry-run` | false | Preview only |
| `--now` | false | Run a full provision-restore-smoke-destroy drill immediately |
| `--install-cron` | false | Install the monthly drill systemd timer |
| `--schedule` | `monthly` | Drill cadence for `--install-cron` (only `monthly` supported) |
| `--hetzner-project` | `""` | Hetzner project name used for drill VM |
| `--vm-type` | `cx22` | Hetzner server type for drill VM |
| `--ssh-key` | `/root/.config/nself/dr-key.pub` | Public SSH key path injected into drill VM |
| `--region` | `fsn1` | Hetzner location for drill VM |
| `--render-cloud-init` | false | Print the cloud-init user-data template and exit |
| `--render-alerts` | false | Print the Prometheus DRDrillFailed rule and exit |

### `dr promote-standby`

| Flag | Default | Description |
|------|---------|-------------|
| `--region` | `""` | Target region for promotion |
| `--yes` | false | Skip confirmation |

### `dr reconfigure-dns`

| Flag | Default | Description |
|------|---------|-------------|
| `--ip` | `""` | New primary IP address (required) |

## Examples

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

## See Also

- [[cmd-backup]] — backup operations
- [[cmd-promote]] — environment promotion
- [[cmd-watchdog]] — self-healing watchdog
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
