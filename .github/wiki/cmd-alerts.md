# nself alerts

> Manage Prometheus alert rules and silences.

## Synopsis

```
nself alerts <subcommand> [flags] [args]
```

## Description

`nself alerts` lists configured Prometheus alert rules, silences alerts for a window, and fires synthetic test alerts to validate routing through Alertmanager. It is the operator-facing wrapper for the monitoring stack provisioned by `nself` (Prometheus + Alertmanager).

`alerts list` reads the bundled rule set and prints each rule with its severity (`P1`, `P2`, `P3`), `for` duration, summary, and PromQL expression. `alerts silence` posts a silence to Alertmanager with a duration and required reason. `alerts test` injects a synthetic firing alert through the Alertmanager API so on-call rotation, notification channels, and dashboards can be exercised end-to-end.

Use this command after editing alert rules or when validating that paging integrations (PagerDuty, Slack, email) are wired correctly.

## Subcommands

| Name | Description |
|------|-------------|
| `list` | List all configured alert rules |
| `silence <alert-name>` | Silence an alert for a specified duration |
| `test <alert-name>` | Fire a synthetic test alert |

## Flags

### `alerts list`

| Flag | Default | Description |
|------|---------|-------------|
| `--severity` | `""` | Filter by severity (`P1`, `P2`, `P3`) |
| `--json` | false | JSON output |

### `alerts silence <alert-name>`

| Flag | Default | Description |
|------|---------|-------------|
| `--duration` | `2h` | Silence duration (e.g. `2h`, `30m`) |
| `--reason` | `""` | Reason for silencing (required) |

### `alerts test <alert-name>`

| Flag | Default | Description |
|------|---------|-------------|
| `--alertmanager-url` | `""` | Alertmanager URL (default: `http://127.0.0.1:9093`) |

## Examples

```bash
# List all alert rules
nself alerts list

# List only critical (P1) rules as JSON
nself alerts list --severity P1 --json

# Silence the HighErrorRate alert for 4 hours during deploy
nself alerts silence HighErrorRate --duration 4h --reason "scheduled deploy"

# Fire a test alert so on-call rotation can verify routing
nself alerts test HighErrorRate
```

## See Also

- [[cmd-monitor]] — monitoring stack management
- [[cmd-watchdog]] — self-healing container watchdog
- [[cmd-health]] — health checks
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
