# nself start

> Boot the ɳSelf stack with health checks and automatic database initialization.

## Synopsis

```
nself start [flags]
nself up [flags]
```

**Alias:** `up`

## Description

`nself start` brings up the entire ɳSelf stack in the correct order: PostgreSQL first, then automatic database initialization (schemas, extensions, permissions), then Hasura, Auth, Nginx, optional services, monitoring, and custom services. Each service is health-checked before the next group starts.

Before launching containers, `nself start` validates that `docker-compose.yml` exists (run `nself build` first), the Docker daemon is running, and all required ports are available. A pre-flight port check scans for conflicts on ports 80, 443, 5432, 8080, 4000, 6379, and 9000 and reports the conflicting process name if a port is in use.

Database initialization is automatic and idempotent, ɳSelf creates the database, schemas (`auth`, `storage`, `public`), and extensions (`pgcrypto`, `citext`, `uuid-ossp`) if they do not already exist. After all services are healthy, the console prints all service URLs.

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
| `--quiet` | false | Suppress progress output (for CI). Preserves `--json` output. |
| `--watch` | false | Enable health auto-restart: poll services and restart unhealthy containers |
| `--skip-plugins` | false | Start base stack only, skipping plugin compose files |
| `--no-monorepo` | false | Disable automatic monorepo backend detection |
| `--allow-legacy` | false | Bypass v0.9 artifact check and proceed with WARNING (not recommended). Use only as a temporary workaround while running `nself migrate`. |
| `--embedded-pg` | false | Boot PostgreSQL via embedded pglite/wasmtime — no Docker postgres container required. pgvector is included. See [[Embedded-Postgres]] for details. |
| `--debug`, `-d` | false | Show debug information |
| `--verbose`, `-v` | false | Show detailed Docker output |
| `--help`, `-h` | — | Show help |

## v0.9 project detection

`nself start` scans the current directory for v0.9 project artifacts before launching any containers. Detection uses five heuristics (v0.9 `docker-compose.yml` header, `NSELF_VERSION=0.x` in `.env`, flat `nginx/` layout, `.nself/config` as a plain file, and `nself.sh` bootstrap script). Two or more hits trigger a hard error:

```
error: v0.9 project detected. Found 3 legacy artifact(s): docker-compose.yml, .env, nginx/nginx.conf
Run `nself migrate` first. See https://docs.nself.org/migrate/from-v0.9
```

A single hit produces a non-blocking warning. Use `nself migrate detect` to see all detected artifacts before running the migration. See [[Upgrade-From-v0.9]] for the full migration guide.

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

# Suppress progress output for CI pipelines
nself start --quiet

# Boot without a Docker postgres container (embedded pglite/wasmtime)
nself start --embedded-pg
```

## First-run transcript

On the very first `nself start` in a project directory, the CLI detects that Docker images have not been pulled yet and streams progress to avoid a silent terminal during the 1-3 minute pull:

```
[1/7] Checking docker-compose.yml ✓
[2/7] Loading configuration ✓
[3/7] Checking port availability ✓
[4/7] Starting PostgreSQL
First run detected — pulling Docker images (this takes 1-3 minutes on slow connections)...
⚙ Pulling images [12s elapsed] — first run takes 1-3 minutes...
⚙ Pulling images [16s elapsed] — first run takes 1-3 minutes...
⚙ Pulling images [24s elapsed] — first run takes 1-3 minutes...
✓ Docker images ready
...
```

The progress line updates every 4 seconds. Once all images are pulled, the marker file `.nself/.first-run-complete` is written and subsequent starts skip the pull step entirely.

Pass `--quiet` to suppress all progress output. Useful in CI where the log is captured elsewhere.

## Aliases

`nself up` is a hidden alias for `nself start`. Same flags. Same behavior. Provided so docker-compose users can keep their muscle memory.

← [[Commands]] | [[Home]] →
