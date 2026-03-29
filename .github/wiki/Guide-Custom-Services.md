# Guide: Custom Services

Add up to 10 custom Docker services (CS_1 through CS_10) to your nSelf stack. Use this for app-specific services that aren't covered by the plugin ecosystem.

## How Custom Services Work

Custom services are defined via environment variables. nSelf reads `CS_1_*` through `CS_10_*` variables and generates the corresponding `docker-compose` service block automatically on `nself build`.

## Adding a Custom Service

### Step 1: Define the service in `.env`

```env
# CS_1: Node.js API server
CS_1_ENABLED=true
CS_1_NAME=my-api
CS_1_IMAGE=node:20-alpine
CS_1_PORT=3100
CS_1_COMMAND=node server.js
CS_1_WORKDIR=/app

# Pass environment variables to the service
CS_1_ENV_NODE_ENV=production
CS_1_ENV_DATABASE_URL=${DATABASE_URL}
```

### Step 2: Rebuild and restart

```bash
nself build    # generates updated docker-compose.yml
nself restart  # applies changes
```

### Step 3: Verify

```bash
nself status          # should show my-api as running
nself urls            # shows URL if routed through nginx
```

## Available Language Templates

See 40+ starter templates to get going quickly:

```bash
nself service templates
```

Templates include: Node.js, Python (FastAPI, Django), Go, Ruby on Rails, PHP, Java Spring Boot, Rust, and more.

## Example: Python Worker (CS_2)

```env
CS_2_ENABLED=true
CS_2_NAME=worker
CS_2_IMAGE=python:3.12-slim
CS_2_COMMAND=python -m celery worker
CS_2_ENV_CELERY_BROKER_URL=redis://redis:6379/0
```

## Custom Service Variables Reference

| Variable | Description |
|----------|-------------|
| `CS_N_ENABLED` | `true` / `false` |
| `CS_N_NAME` | Service name (lowercase, alphanumeric + hyphens) |
| `CS_N_IMAGE` | Docker image reference |
| `CS_N_PORT` | Internal port the service listens on |
| `CS_N_COMMAND` | Override container command |
| `CS_N_WORKDIR` | Working directory inside container |
| `CS_N_ENV_*` | Environment variables passed to the service |

## See Also

- [[Config-Custom-Services]] — full env var reference
- [[cmd-service]] — service command reference
- [[Architecture]] — how custom services fit the stack

---
← [[Home]] | [[_Sidebar]]
