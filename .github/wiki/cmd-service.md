# nself service

> Enable, disable, and list optional nSelf services.

## Synopsis

```
nself service <subcommand> [flags]
```

## Description

`nself service` manages the optional services in your nSelf stack. The four core services (PostgreSQL, Hasura, Auth, Nginx) are always included. Everything else — Redis, MinIO, Mailpit, Functions, MeiliSearch, MLflow, Monitoring, and Admin — is optional and controlled through this command.

Enabling or disabling a service writes to your `.env` file. After changing service state, run `nself build` to regenerate `docker-compose.yml` with the updated service set, then `nself restart` to apply the changes.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `enable <name>` | Enable an optional service |
| `disable <name>` | Disable an optional service |
| `list` | List all optional services with enabled/disabled status |

## Available Services

| Service | Env Var | Aliases |
|---------|---------|---------|
| `redis` | `REDIS_ENABLED` | — |
| `minio` | `MINIO_ENABLED` | `storage` |
| `mailpit` | `MAILPIT_ENABLED` | `email` |
| `functions` | `FUNCTIONS_ENABLED` | — |
| `search` | `SEARCH_ENABLED` | `meilisearch` |
| `mlflow` | `MLFLOW_ENABLED` | — |
| `monitoring` | `MONITORING_ENABLED` | — |
| `admin` | `NSELF_ADMIN_ENABLED` | — |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--env` | current | Target environment file |
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

# Enable monitoring for production environment
nself service enable monitoring --env prod

# Disable Mailpit
nself service disable mailpit

# Disable MeiliSearch (using alias)
nself service disable meilisearch

# After changing services, rebuild and restart
nself build && nself restart
```

← [[Commands]] | [[Home]] →
