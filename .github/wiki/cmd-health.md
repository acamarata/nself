# nself health

<!-- BEGIN PROSE:summary -->
> Health check management with continuous monitoring.
<!-- END PROSE:summary -->

## Synopsis

```
nself health [subcommand] <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself health` runs health checks against running ɳSelf services and HTTP endpoints. Running `nself health` without a subcommand executes all health checks, the same as `nself health check`.

Each service is checked using its native health method: `pg_isready` for PostgreSQL, HTTP `/healthz` for Hasura and Auth, `PING` for Redis, and `/health` for Nginx. Response times are shown alongside the health status.

The `watch` subcommand provides continuous monitoring, it re-runs all checks every `--interval` seconds until you press Ctrl+C. Use `--quiet` to suppress output when all services are healthy and only print on failure, suitable for monitoring scripts.


A service that declares no Docker healthcheck reports as `running` rather than `healthy`, and counts as healthy. Only some services define a healthcheck, so treating `running` as a failure would mark most of a working stack unhealthy. `--quiet` stays quiet for those services, and `--json` counts them in the healthy total.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--env` | `""` | Environment to load config for |
| `--interval` | `10` | Watch interval in seconds |
| `--json` | `false` | Output in JSON format |
| `--quiet` | `false` | Only output on failure |
| `--retries` | `3` | Retry count on failure |
| `--timeout` | `30` | Check timeout in seconds |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `check` | Run all health checks |
| `config` | Show health check settings |
| `endpoint` | Check an HTTP endpoint |
| `history` | Show last 20 health checks |
| `service` | Check a single service |
| `watch` | Continuous health monitoring |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Run all health checks
nself health
nself health check

# Check a single service
nself health service postgres
nself health service hasura

# Check an HTTP endpoint
nself health endpoint https://api.myapp.dev/health

# Continuous monitoring every 5 seconds
nself health watch --interval 5

# Monitor silently — only print on failure
nself health watch --quiet

# Show last 20 health check results
nself health history

# Show health check configuration
nself health config

# JSON output for monitoring integrations
nself health --json
```

**Sample output:**

```
Service              Status       Time     Details
postgres             ✓ healthy    3ms      pg_isready
hasura               ✓ healthy    45ms     /healthz 200
auth                 ✓ healthy    52ms     /healthz 200
nginx                ✓ healthy    8ms      /health 200
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
