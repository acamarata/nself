# nself monitor

> Monitoring stack management.

## Synopsis

```
nself monitor <subcommand> [flags]
```

## Description

`nself monitor` manages the monitoring stack provisioned with the project (Prometheus, Grafana, Loki, Alertmanager, exporters). Right now it focuses on the Grafana dashboard library: re-provisioning the bundled dashboards from the latest templates so a long-running stack can pick up dashboard updates that ship with newer CLI versions.

`monitor upgrade-dashboards` re-provisions the eleven nSelf-bundled Grafana dashboards: System Overview, Postgres, Hasura, Nginx, Per-Plugin, Request Latency, AI Cost Tracker, User Activity, Error Heatmap, Backups, Licenses. By default, dashboards that are already current are left alone. Pass `--force` to re-provision unconditionally.

For the running monitoring services (start/stop/health), use `nself service` and `nself status`. For alert rule and silence management, use `nself alerts`.

## Subcommands

| Name | Description |
|------|-------------|
| `upgrade-dashboards` | Upgrade Grafana dashboards to the latest bundled versions |

## Flags

### `monitor upgrade-dashboards`

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | false | Force re-provision even if up to date |

## Examples

```bash
# Re-provision dashboards that have changed since last run
nself monitor upgrade-dashboards

# Force re-provision every dashboard
nself monitor upgrade-dashboards --force
```

## See Also

- [[cmd-alerts]] — alert rules and silences
- [[cmd-service]] — manage optional services
- [[cmd-status]] — service health
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
