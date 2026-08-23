# nself watchdog

<!-- BEGIN PROSE:summary -->
> Self-healing container watchdog with circuit breaker.
<!-- END PROSE:summary -->

## Synopsis

```
nself watchdog <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself watchdog` is the operator interface to the self-healing container watchdog. The watchdog monitors service containers, restarts unhealthy ones, and tracks failures with a circuit breaker per service to avoid restart storms. When a circuit trips, the watchdog stops auto-restarting that service and emits an alert; an operator must reset the breaker after fixing the root cause.

`watchdog status` shows whether the watchdog daemon is running and prints the per-service circuit breaker state. `watchdog reset-breakers` resets all tripped breakers to closed. `watchdog history` prints recent watchdog events (restarts, breaker trips). `watchdog test-alert` synthesizes an alert through every configured channel (Telegram bot, SMTP) so on-call rotation can verify routing without waiting for a real failure.

`--since` accepts Go duration syntax (`24h`, `7d`). JSON output is suitable for shipping into a metrics pipeline.

### `watchdog status`
### `watchdog history`
### `watchdog test-alert`
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
| `history` | Show watchdog event history |
| `reset` | Reset a specific service circuit breaker (including PERMANENT_OPEN) |
| `reset-breakers` | Reset all tripped circuit breakers to closed state |
| `status` | Show watchdog status and circuit breaker states |
| `test-alert` | Send a test alert through all configured channels (TG + email) |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
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
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-alerts]], Prometheus alert rules
- [[cmd-health]], health checks
- [[cmd-status]], service health
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
