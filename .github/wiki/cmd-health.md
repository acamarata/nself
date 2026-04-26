# nself health

> Health check management with continuous monitoring.

## Synopsis

```
nself health [subcommand] [flags]
```

## Description

`nself health` runs health checks against running ɳSelf services and HTTP endpoints. Running `nself health` without a subcommand executes all health checks, the same as `nself health check`.

Each service is checked using its native health method: `pg_isready` for PostgreSQL, HTTP `/healthz` for Hasura and Auth, `PING` for Redis, and `/health` for Nginx. Response times are shown alongside the health status.

The `watch` subcommand provides continuous monitoring, it re-runs all checks every `--interval` seconds until you press Ctrl+C. Use `--quiet` to suppress output when all services are healthy and only print on failure, suitable for monitoring scripts.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `check` | Run all health checks (default when no subcommand given) |
| `service <name>` | Check a single service by name |
| `endpoint <url>` | Check an HTTP endpoint |
| `watch` | Continuous health monitoring (Ctrl+C to stop) |
| `history` | Show last 20 health checks |
| `config` | Show health check settings |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--timeout` | `30` | Check timeout in seconds |
| `--interval` | `10` | Watch interval in seconds (for `watch`) |
| `--retries` | `3` | Retry count on failure |
| `--json` | false | JSON output |
| `--quiet` | false | Only output on failure |
| `--env` | `""` | Environment to load config for |
| `--help`, `-h` | — | Show help |

## Examples

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

← [[Commands]] | [[Home]] →
