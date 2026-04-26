# nself status

> Show health status of all services.

## Synopsis

```
nself status [SERVICE] [flags]
```

## Description

`nself status` displays the running/stopped state of every service in the ɳSelf stack. For each service it shows the health status (healthy, unhealthy, starting), container name, and response time from the health endpoint. A summary line at the bottom shows the overall healthy count.

You can pass a single service name to show detailed status for that service only. Use `--verbose` to include resource usage (CPU, memory) and uptime. Use `--json` to get machine-readable output suitable for monitoring scripts or dashboards.

Exit codes are meaningful: `0` means all services are healthy, `1` means an error occurred running the checks, and `2` means one or more services are unhealthy.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--json`, `-j` | false | JSON output |
| `--verbose` | false | Show resource usage and uptime |
| `--metrics` | false | Show performance metrics |
| `--health-only` | false | Show only health status, no other details |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Show status of all services
nself status

# Show status for a specific service
nself status postgres

# JSON output for scripting
nself status --json

# Include resource usage and uptime
nself status --verbose

# Show performance metrics
nself status --metrics
```

**Sample output:**

```
Service         Status      Details
postgres        ✓ healthy   pg_isready (10ms)
hasura          ✓ healthy   /healthz 200 (45ms)
auth            ✓ healthy   /healthz 200 (52ms)
nginx           ✓ healthy   /health 200 (8ms)
redis           ✓ healthy   PONG (3ms)

Summary: 5/5 services healthy
```

← [[Commands]] | [[Home]] →
