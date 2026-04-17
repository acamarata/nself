# nself start

> Boot the nSelf stack with health checks and automatic database initialization.

## Synopsis

```
nself start [flags]
nself up [flags]
```

**Alias:** `up`

## Description

`nself start` brings up the entire nSelf stack in the correct order: PostgreSQL first, then automatic database initialization (schemas, extensions, permissions), then Hasura, Auth, Nginx, optional services, monitoring, and custom services. Each service is health-checked before the next group starts.

Before launching containers, `nself start` validates that `docker-compose.yml` exists (run `nself build` first), the Docker daemon is running, and all required ports are available. A pre-flight port check scans for conflicts on ports 80, 443, 5432, 8080, 4000, 6379, and 9000 and reports the conflicting process name if a port is in use.

Database initialization is automatic and idempotent — nSelf creates the database, schemas (`auth`, `storage`, `public`), and extensions (`pgcrypto`, `citext`, `uuid-ossp`) if they do not already exist. After all services are healthy, the console prints all service URLs.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--fresh` | false | Force recreate all containers |
| `--force-recreate` | false | Alias for `--fresh` |
| `--clean-start` | false | Remove all containers before starting |
| `--skip-build` | false | Skip automatic rebuild detection |
| `--skip-health-checks` | false | Skip health validation after startup |
| `--skip-port-check` | false | Skip port availability check |
| `--quick` | false | Quick start (timeout=30s, required=60%) |
| `--timeout` | `120` | Health check timeout in seconds (range: 30–600) |
| `--no-monorepo` | false | Disable automatic monorepo backend detection |
| `--debug`, `-d` | false | Show debug information |
| `--verbose`, `-v` | false | Show detailed Docker output |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Standard boot
nself start

# Using the alias
nself up

# Force container recreation (useful after config changes)
nself start --fresh

# Remove existing containers before starting fresh
nself start --clean-start

# Fast mode for CI — lower timeout, 60% health threshold
nself start --quick

# Skip health checks (not recommended for production)
nself start --skip-health-checks

# Custom health check timeout
nself start --timeout 300

# Verbose output to debug startup issues
nself start -v
```

## Aliases

`nself up` is a hidden alias for `nself start`. Same flags. Same behavior. Provided so docker-compose users can keep their muscle memory.

← [[Commands]] | [[Home]] →
