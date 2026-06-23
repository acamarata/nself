# nself service

> Enable, disable, and manage optional ɳSelf services, and scaffold custom services into your project.

## Synopsis

```
nself service <subcommand> [flags]
```

## Description

`nself service` manages the optional services in your ɳSelf stack. The four core services (PostgreSQL, Hasura, Auth, Nginx) are always included. Six optional services, Redis, MinIO, Email, Functions, Search, and Admin, are toggled through this command.

Enabling or disabling a service writes to your `.env` file. After changing service state, run `nself build` to regenerate `docker-compose.yml` with the updated service set, then `nself restart` to apply the changes.

`nself service add` scaffolds a new custom service into a free `CS_N` slot (1-10). The `--template` flag controls the language starter that is emitted.

**MLflow** is no longer an optional service. Use `nself plugin install mlflow` instead.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `enable <name>` | Enable an optional service |
| `disable <name>` | Disable an optional service |
| `list` | List all optional services with enabled/disabled status |
| `upgrade <name> <version>` | Pin a service to a specific image version |
| `configure <service>` | Configure a service (e.g. email provider presets) |
| `add <name>` | Scaffold a new custom service into the next free `CS_N` slot |
| `start <name>` | Start a named service without rebuilding the full stack |
| `stop <name>` | Stop a named service (container preserved) |
| `restart <name>` | Restart a named service |
| `ps` | Show live status of all services in the running stack |
| `update <name>` | Pull the latest image for a service and restart it |
| `scale <name> <replicas>` | Set the replica count for a named service |

## Available Optional Services (7)

| Service | Env Var | Default Port | Aliases |
|---------|---------|--------------|---------|
| `redis` | `REDIS_ENABLED` | 6379 | — |
| `minio` | `MINIO_ENABLED` | 9000 | `storage` |
| `email` | `MAILPIT_ENABLED` | 8025 | `mail`, `mailpit` |
| `functions` | `FUNCTIONS_ENABLED` | 3008 | — |
| `search` | `SEARCH_ENABLED` | 7700 | `meilisearch` |
| `monitoring` | `MONITORING_ENABLED` | — | — |
| `admin` | `NSELF_ADMIN_ENABLED` | 3021 | — |

## Custom Services (`service add`)

`nself service add` scaffolds a new service into a free `CS_N` slot (up to 10). Use `--template` to choose the language starter.

### Supported templates

| Template | Description |
|----------|-------------|
| `go` (default) | Go HTTP service with `main.go` and `Dockerfile` |
| `node` | Node.js service with `index.js` and `Dockerfile` |
| `python` | Python (Flask/FastAPI) service with `main.py` and `Dockerfile` |
| `static` | Static file server (nginx-based) with placeholder `index.html` |
| `rust` | Rust HTTP service with `main.rs` and `Cargo.toml` |
| `other` | Generic Dockerfile with minimal scaffold |

### Flags for `service add`

| Flag | Default | Description |
|------|---------|-------------|
| `--template` | `go` | Language template: `go`, `node`, `python`, `static`, `rust`, `other` |
| `--lang` | — | Hidden backward-compat alias for `--template` |
| `--force` | false | Overwrite an existing service directory |
| `--dry-run` | false | Print what would be done without writing any files |

After scaffolding, run `nself build` to include the new service in `docker-compose.yml`.

## Flags (common)

| Flag | Default | Description |
|------|---------|-------------|
| `--env` | current | Target environment file (reads `.env.{env}`) |
| `--json` | false | Output as JSON array (for `list`) |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# List all optional services
nself service list
nself service list --json

# Enable Redis
nself service enable redis

# Enable MinIO object storage (using alias)
nself service enable storage

# Enable monitoring stack
nself service enable monitoring

# Enable monitoring for the staging environment
nself service enable monitoring --env staging

# Disable Mailpit
nself service disable mailpit

# Disable MeiliSearch (using alias)
nself service disable meilisearch

# After changing services, rebuild and restart
nself build && nself restart

# Scaffold a new Go custom service
nself service add myapi

# Scaffold a Python custom service
nself service add myapi --template python

# Scaffold with dry-run to preview what would be written
nself service add myapi --template node --dry-run

# Use the legacy --lang alias (same as --template)
nself service add myapi --lang rust

# Start/stop/restart a named service without a full rebuild
nself service start redis
nself service stop minio
nself service restart hasura

# Show live status of all stack services
nself service ps

# Pull the latest image and restart
nself service update hasura

# Scale the functions service to 3 replicas
nself service scale functions 3

# Pin postgres to a specific version
nself service upgrade postgres 16.3
```

## See Also

- [[cmd-build]] — regenerate `docker-compose.yml` after changing service state
- [[cmd-start]] — boot the full nSelf stack
- [[cmd-doctor]] — diagnose service health issues
- [[cmd-plugin]] — manage nSelf plugins (e.g. `mlflow`)

← [[Commands]] | [[Home]] →
