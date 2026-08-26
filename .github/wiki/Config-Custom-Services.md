# Config, Custom Services

Custom services are user-defined Docker containers that ɳSelf manages alongside the core stack. They live in numbered slots, `CS_1` through `CS_10`, and are treated as first-class citizens: ɳSelf generates their `docker-compose` config, injects core service credentials, routes them through Nginx, and exposes them in `nself status` and `nself logs`.

Use custom services to run any application your project needs, a REST API, a background worker, a webhook processor, an AI inference service, without leaving the ɳSelf config system.

---

## Overview

```
nSelf stack
  ├── postgres, hasura, auth, nginx  (core services — always present)
  ├── redis, minio, search, ...      (optional services — toggled by *_ENABLED)
  └── CS_1 .. CS_10                  (custom services — defined in .env)
        ├── any Docker image or nself template
        ├── auto-receives DATABASE_URL, HASURA_GRAPHQL_ADMIN_SECRET
        ├── optionally exposed via Nginx at {CS_N_ROUTE}.{BASE_DOMAIN}
        └── managed by nself start / stop / logs / status
```

Up to **10 custom service slots** are available per project. Slots are independent, you can use `CS_1` and `CS_3` without defining `CS_2`.

---

## Defining a Custom Service

### Option 1, Shorthand (CS_N)

The `CS_N` variable accepts a single definition string that encodes the most common settings:

```
CS_N=name:template[:port][:route]
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Service name. Used in container labels, logs, and Nginx config. |
| `template` | Yes | Language/framework template. See [Language Templates](#language-templates). |
| `port` | No | Port the service listens on. Auto-assigned if omitted. |
| `route` | No | Nginx subdomain. If omitted, the service is internal-only. |

**Example, a Go service on port 8001 exposed at `ping.{BASE_DOMAIN}`:**

```bash
# .env
CS_1=ping_api:go:8001:ping
```

ɳSelf parses this and derives the individual `CS_1_*` variables automatically. You only need the individual vars if you want to override a specific field.

### Option 2, Individual Variables

For full control, set each variable explicitly. Individual vars always take precedence over parsed values from `CS_N`.

```bash
# .env
CS_1=ping_api:go:8001:ping   # base definition
CS_1_MEMORY=512m             # override the default 256m
CS_1_CPU=1.0                 # override the default 0.5
CS_1_PUBLIC=true             # expose via Nginx
CS_1_REPLICAS=2              # run two instances
```

---

## Environment Variable Reference

All variables use the pattern `CS_N_*` where `N` is the slot number (1–10). Variables marked as "parsed from CS_N" are derived automatically when the shorthand form is used.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `CS_N` | string | *(empty)* | Shorthand definition. Format: `name:template[:port][:route]` |
| `CS_N_NAME` | string | parsed from CS_N | Service name used in container labels and Nginx routing |
| `CS_N_TEMPLATE` | string | parsed from CS_N | Framework template (see [Language Templates](#language-templates)) |
| `CS_N_PORT` | int | parsed from CS_N or auto | Port the service listens on inside the Docker network |
| `CS_N_ROUTE` | string | parsed from CS_N | Nginx subdomain , set this to expose the service externally |
| `CS_N_PUBLIC` | bool | `false` | When `true`, Nginx routes `{CS_N_ROUTE}.{BASE_DOMAIN}` to this service |
| `CS_N_MEMORY` | string | `256m` | Docker memory limit |
| `CS_N_CPU` | string | `0.5` | Docker CPU limit (fractional cores) |
| `CS_N_REPLICAS` | int | `1` | Number of container instances to run |
| `CS_N_HEALTHCHECK` | string | `/health` | Health endpoint path , used by Docker healthcheck and `nself status` |
| `CS_N_TABLE_PREFIX` | string | *(empty)* | Database table prefix for this service's migrations |
| `CS_N_ENV` | string | *(empty)* | Additional env vars to inject, in `KEY=VALUE,KEY=VALUE` format |

All `CS_*` variables are automatically exempt from "unknown env var" warnings.

---

## Env Var Injection

When a custom service container starts, ɳSelf automatically injects the following variables:

| Variable | Value | Description |
|----------|-------|-------------|
| `DATABASE_URL` | `postgresql://user:pass@postgres:5432/db` | Full PostgreSQL connection string |
| `HASURA_GRAPHQL_ADMIN_SECRET` | from project `.env` | For making privileged Hasura API calls |
| `PROJECT_NAME` | from project `.env` | The project namespace |
| `BASE_DOMAIN` | from project `.env` | The root domain |
| `ENV` | from project `.env` | `dev`, `staging`, or `prod` |
| `CS_N_ENV` values | parsed from `CS_N_ENV` | Any additional vars you define for this slot |

All variables from the project's `.env` that do not conflict with injected vars are also forwarded into the container. This means your service can read any project-level config it needs without extra wiring.

**Example, reading injected vars in a Go service:**

```go
dbURL := os.Getenv("DATABASE_URL")            // injected by nSelf
adminSecret := os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET") // injected by nSelf
apiKey := os.Getenv("MY_API_KEY")             // forwarded from .env
```

### Adding Extra Vars

Use `CS_N_ENV` to pass service-specific variables that aren't in the main `.env`:

```bash
# .env
CS_2_ENV=QUEUE_SIZE=100,WORKER_TIMEOUT=30,LOG_FORMAT=json
```

---

## Language Templates

`nself service add <name> --template <lang>` scaffolds a production-ready service with a Dockerfile, health endpoint, and example database connection.

```bash
nself service add my-api --template go
```

**Available templates:**

| Template | Description |
|----------|-----------|
| `go` (default) | Go HTTP service |
| `node` | Node.js service |
| `python` | Python service |
| `static` | Static file server |
| `rust` | Rust service |
| `other` | Minimal scaffold for anything else |

Each template includes a `/health` endpoint that returns `200 OK`, which ɳSelf uses for container health checks and `nself status`.

---

## Real-World Example, ping_api (web/backend CS_1)

The ɳSelf infrastructure itself (`web/backend`) uses `CS_1` to run **ping_api**, a Go service on port 8001 that handles telemetry and license validation for the CLI. It is accessible at `ping.nself.org`.

```bash
# web/backend/.env
CS_1=ping_api:go:8001:ping
CS_1_PUBLIC=true
CS_1_MEMORY=128m
CS_1_CPU=0.25
CS_1_REPLICAS=1
CS_1_HEALTHCHECK=/health
```

This configuration produces:

| Derived value | Result |
|---------------|--------|
| `CS_1_NAME` | `ping_api` |
| `CS_1_TEMPLATE` | `go` |
| `CS_1_PORT` | `8001` |
| `CS_1_ROUTE` | `ping` |
| External URL | `https://ping.nself.org` |
| Internal URL | `http://ping_api:8001` (within Docker network) |

The ping_api service receives `DATABASE_URL` and `HASURA_GRAPHQL_ADMIN_SECRET` automatically on startup, so it can validate license keys against the database without any additional configuration.

---

## Quick Start

**1. Create a service using a template:**

```bash
nself service add my-api --template go
```

This scaffolds a Go service in `./services/my-api/` with a Dockerfile and `/health` endpoint, and assigns it the next available `CS_N` slot and port automatically.

**2. Register it in `.env`:**

```bash
# .env
CS_2=my_api:go:8002:api
CS_2_PUBLIC=true
CS_2_MEMORY=256m
```

**3. Rebuild and start:**

```bash
nself build     # regenerates docker-compose.yml with CS_2 included
nself start     # brings up the full stack including my_api
```

**4. Verify it's running:**

```bash
nself status    # shows all services including custom ones
nself logs cs2  # tail logs for the CS_2 container
```

Your service is now live at `https://api.{BASE_DOMAIN}` and accessible internally at `http://my_api:8002`.

---

## Multiple Custom Services

You can define up to 10 slots simultaneously. Slots are independent, gaps are fine.

```bash
# .env
CS_1=ping_api:go:8001:ping           # telemetry service
CS_2=worker:node:8002                 # background job worker (internal-only, no route)
CS_3=cms_api:python:8003:cms          # FastAPI CMS
CS_5=search_proxy:go:8005:search      # search proxy (skipped CS_4 intentionally)

# Override resources on the worker
CS_2_MEMORY=512m
CS_2_CPU=1.0
CS_2_REPLICAS=3
```

---

## Notes

- Custom service images are built from the scaffolded Dockerfile in `./services/{name}/`. To use a pre-built image instead, set `CS_N_IMAGE` directly (advanced usage, see [[Guide-Custom-Services]]).
- The `CS_N_TABLE_PREFIX` variable is used by `nself migrate` to scope migrations to a subdirectory, keeping custom service migrations separate from core schema changes.
- Custom services participate in `nself backup`, the backup bundle includes a dump of any tables matching the `CS_N_TABLE_PREFIX`.
- Logs from all custom service slots are included in `nself logs --all`.

---

For a step-by-step walkthrough of building and deploying a custom service from scratch, see [[Guide-Custom-Services]].

← [[Config-Optional-Services]] | [[Configuration]] | [[Guide-Custom-Services]] →
