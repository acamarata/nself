# nself watchdog

> Self-healing container watchdog with circuit breaker.

## Synopsis

```
nself watchdog <subcommand> [flags]
```

## Description

`nself watchdog` is the operator interface to the self-healing container watchdog. The watchdog monitors service containers, restarts unhealthy ones, and tracks failures with a circuit breaker per service to avoid restart storms. When a circuit trips, the watchdog stops auto-restarting that service and emits an alert; an operator must reset the breaker after fixing the root cause.

`watchdog status` shows whether the watchdog daemon is running and prints the per-service circuit breaker state. `watchdog reset-breakers` resets all tripped breakers to closed. `watchdog history` prints recent watchdog events (restarts, breaker trips). `watchdog test-alert` synthesizes an alert through every configured channel (Telegram bot, SMTP) so on-call rotation can verify routing without waiting for a real failure.

`--since` accepts Go duration syntax (`24h`, `7d`). JSON output is suitable for shipping into a metrics pipeline.

## Subcommands

| Name | Description |
|------|-------------|
| `status` | Show watchdog status and circuit breaker states |
| `reset-breakers` | Reset all tripped circuit breakers to closed state |
| `history` | Show watchdog event history |
| `test-alert` | Send a test alert through all configured channels |

## Flags

### `watchdog status`

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | JSON output |

### `watchdog history`

| Flag | Default | Description |
|------|---------|-------------|
| `--since` | `24h` | Show events since duration (e.g. `24h`, `7d`) |
| `--json` | false | JSON output |

### `watchdog test-alert`

| Flag | Default | Description |
|------|---------|-------------|
| `--service` | `test-service` | Service name for the test alert |
| `--severity` | `critical` | Severity level (`warning`, `critical`) |

## Examples

```bash
# Show current status with circuit breaker states
nself watchdog status

# Same as JSON for a metrics pipeline
nself watchdog status --json

# Reset all tripped breakers after a manual fix
nself watchdog reset-breakers

# See watchdog events from the past week
nself watchdog history --since 7d

# Verify alert routing
nself watchdog test-alert --severity warning
```

## See Also

- [[cmd-alerts]] — Prometheus alert rules
- [[cmd-health]] — health checks
- [[cmd-status]] — service health
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
