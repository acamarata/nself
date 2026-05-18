# Config: Environment Variables

← [[Configuration]] | [[Config-Postgres]] →

---

All ɳSelf project configuration lives in `.env` (and optionally `.env.local` for local overrides). This page documents every supported environment variable, grouped by service.

**Auto-generated vars:** A set of derived variables (e.g. `DATABASE_URL`, `DOCKER_NETWORK`) are written to `.env.computed` by the orchestration layer. Never hand-edit `.env.computed`, it is overwritten on every `nself build` and `nself start`.

**Plugin-managed vars:** Plugins that require additional configuration (e.g. `nself-ai`, `nself-notify`, `nself-livekit`) inject their own `*_ENABLED`, `*_KEY`, and `*_PORT` vars into `.env` when installed. Those are documented in each plugin's own wiki page. Only core CLI vars appear here.

---

## Table of Contents

- [Core Project Settings](#core-project-settings)
- [PostgreSQL](#postgresql)
- [Hasura (GraphQL API)](#hasura-graphql-api)
- [Auth](#auth)
- [Nginx and SSL](#nginx-and-ssl)
- [ɳSelf Admin](#nself-admin)
- [Optional Service Toggles](#optional-service-toggles)
- [Custom Services (CS\_N)](#custom-services-cs_n)
- [Computed Variables](#computed-variables)

---

## Core Project Settings

These vars control the top-level identity and behavior of a project.

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `PROJECT_NAME` | string | *(none)* | **Yes** | Docker container and network namespace. Must be lowercase, 2–30 characters. |
| `BASE_DOMAIN` | string | `local.nself.org` | No | Root domain used to construct all service subdomains. |
| `ENV` | enum | `dev` | No | Deployment environment. Accepted values: `dev`, `staging`, `prod`. |
| `PROJECT_DESCRIPTION` | string | `""` | No | Human-readable description of the project. |
| `ADMIN_EMAIL` | string | `""` | No | Admin contact email for notifications and certificates. |
| `DB_ENV_SEEDS` | bool | `true` | No | When `true`, database seed files run automatically on first boot. |

---

## PostgreSQL

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `POSTGRES_VERSION` | string | `16-alpine` | No | Docker image tag for the Postgres container. |
| `POSTGRES_HOST` | string | `postgres` | No | Internal container hostname. Change only if running Postgres externally. |
| `POSTGRES_PORT` | int | `5432` | No | Port exposed to the host machine. |
| `POSTGRES_DB` | string | `nself` | No | Name of the default database created on first start. |
| `POSTGRES_USER` | string | `postgres` | No | Superuser account name. |
| `POSTGRES_PASSWORD` | string | *(none)* | **Yes** | Superuser password. Minimum 16 characters. |
| `POSTGRES_EXTENSIONS` | string | `uuid-ossp` | No | Comma-separated list of extensions to install automatically. |
| `POSTGRES_EXPOSE_PORT` | enum | `auto` | No | Controls host-port binding. `auto` exposes in dev and hides in prod. Accepted values: `auto`, `true`, `false`. |
| `POSTGRES_MEM_LIMIT` | string | `2g` | No | Docker memory limit for the Postgres container. |
| `POSTGRES_CPU_LIMIT` | string | `2.0` | No | Docker CPU core limit for the Postgres container. |

**Computed:** `DATABASE_URL` is derived automatically:

```
DATABASE_URL=postgresql://{POSTGRES_USER}:{POSTGRES_PASSWORD}@postgres:5432/{POSTGRES_DB}
```

This value is written to `.env.computed` and should not be set manually.

---

## Hasura (GraphQL API)

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `HASURA_VERSION` | string | `v2.44.0` | No | Docker image tag for Hasura. |
| `HASURA_GRAPHQL_ADMIN_SECRET` | string | *(none)* | **Yes** | Admin API secret used to authenticate privileged requests. Minimum 32 characters. |
| `HASURA_JWT_KEY` | string | *(none)* | **Yes** | Secret key used to sign and verify JWTs. Minimum 32 characters. |
| `HASURA_JWT_TYPE` | string | `HS256` | No | JWT signing algorithm. Common values: `HS256`, `RS256`. |
| `HASURA_GRAPHQL_ENABLE_CONSOLE` | bool | `true` (dev) / `false` (prod) | No | Enable the Hasura web console. Automatically disabled in production. |
| `HASURA_GRAPHQL_DEV_MODE` | bool | `true` (dev) / `false` (prod) | No | Enable developer mode. Automatically disabled in production. |
| `HASURA_GRAPHQL_CORS_DOMAIN` | string | `http://localhost:*` (dev) | No | Allowed CORS origins. In production, set to your actual domain(s). |
| `HASURA_GRAPHQL_LOG_LEVEL` | string | `warn` | No | Hasura log verbosity. Accepted values: `debug`, `info`, `warn`, `error`. |
| `HASURA_PORT` | int | `8080` | No | Internal port the Hasura container listens on. |
| `HASURA_ROUTE` | string | `api` | No | Nginx subdomain route (e.g. `api` → `api.yourdomain.com`). |
| `HASURA_MEM_LIMIT` | string | `1g` | No | Docker memory limit for the Hasura container. |
| `HASURA_CPU_LIMIT` | string | `1.0` | No | Docker CPU core limit for the Hasura container. |

---

## Auth

Authentication is provided by nHost Auth. These variables configure the Auth service container.

### Core Auth

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `AUTH_VERSION` | string | `0.36.0` | No | Docker image tag for the Auth service. |
| `AUTH_PORT` | int | `4000` | No | Internal port the Auth service listens on. |
| `AUTH_CLIENT_URL` | string | `http://localhost:3000` | No | URL the Auth service redirects to after OAuth flows complete. |
| `AUTH_ACCESS_TOKEN_EXPIRES_IN` | int | `900` | No | Access token lifetime in seconds (default: 15 minutes). |
| `AUTH_REFRESH_TOKEN_EXPIRES_IN` | int | `2592000` | No | Refresh token lifetime in seconds (default: 30 days). |
| `AUTH_RATE_LIMIT` | string | `30r/m` | No | Nginx rate limit applied to auth endpoints. |
| `AUTH_MEM_LIMIT` | string | `256m` | No | Docker memory limit for the Auth container. |
| `AUTH_CPU_LIMIT` | string | `0.25` | No | Docker CPU core limit for the Auth container. |
| `AUTH_LOG_LEVEL` | string | `info` | No | Auth service log verbosity. Accepted values: `debug`, `info`, `warn`, `error`. |

### SMTP (for Auth emails)

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `AUTH_SMTP_HOST` | string | `mailpit` | No | SMTP server hostname. Defaults to the local Mailpit container in dev. |
| `AUTH_SMTP_PORT` | int | `1025` | No | SMTP server port. |
| `AUTH_SMTP_USER` | string | *(empty)* | No | SMTP authentication username. |
| `AUTH_SMTP_PASS` | string | *(empty)* | No | SMTP authentication password. |
| `AUTH_SMTP_SECURE` | bool | `false` | No | When `true`, connect using TLS. |
| `AUTH_SMTP_SENDER` | string | `noreply@{BASE_DOMAIN}` | No | From address for outgoing auth emails. |

### OAuth Providers

All OAuth provider vars are optional. Set client ID and secret for each provider you want to enable.

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `AUTH_PROVIDER_GOOGLE_CLIENT_ID` | string | *(empty)* | No | Google OAuth 2.0 client ID. |
| `AUTH_PROVIDER_GOOGLE_CLIENT_SECRET` | string | *(empty)* | No | Google OAuth 2.0 client secret. |
| `AUTH_PROVIDER_GITHUB_CLIENT_ID` | string | *(empty)* | No | GitHub OAuth app client ID. |
| `AUTH_PROVIDER_GITHUB_CLIENT_SECRET` | string | *(empty)* | No | GitHub OAuth app client secret. |
| `AUTH_PROVIDER_APPLE_CLIENT_ID` | string | *(empty)* | No | Apple Sign In service ID. |
| `AUTH_PROVIDER_FACEBOOK_CLIENT_ID` | string | *(empty)* | No | Facebook app client ID. |

---

## Nginx and SSL

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `NGINX_VERSION` | string | `alpine` | No | Docker image tag for Nginx. |
| `NGINX_HTTP_PORT` | int | `80` | No | Host port for HTTP traffic. |
| `NGINX_HTTPS_PORT` | int | `443` | No | Host port for HTTPS traffic. |
| `NGINX_BIND_IP` | string | `127.0.0.1` (dev) / `0.0.0.0` (prod) | No | IP address Nginx binds on. Set to `0.0.0.0` to accept external connections. |
| `NGINX_CLIENT_MAX_BODY_SIZE` | string | `100M` | No | Maximum allowed size for client request bodies (controls upload limits). |
| `SSL_MODE` | enum | `local` | No | SSL certificate strategy. Accepted values: `local` (self-signed), `letsencrypt`, `custom`, `none`. |
| `EXTRA_SSL_DOMAINS` | string | *(empty)* | No | Additional domains to include in the certificate SAN (comma-separated). |

---

## ɳSelf Admin

The ɳSelf Admin dashboard is an optional local GUI companion that runs at `localhost:3021`. It is not deployed to any server.

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `NSELF_ADMIN_ENABLED` | bool | `false` | No | When `true`, the Admin container is included in `docker-compose`. |
| `NSELF_ADMIN_PORT` | int | `3021` | No | Host port for the Admin dashboard. |
| `NSELF_ADMIN_VERSION` | string | `latest` | No | Docker image tag for `nself/nself-admin`. |

---

## Optional Service Toggles

These boolean flags enable optional bundled services. Each defaults to `false`. When set to `true`, the service is included in the generated `docker-compose.yml`.

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `REDIS_ENABLED` | bool | `false` | No | Enable Redis (caching, sessions, queues). |
| `MINIO_ENABLED` | bool | `false` | No | Enable MinIO (S3-compatible object storage). |
| `SEARCH_ENABLED` | bool | `false` | No | Enable MeiliSearch (full-text search). |
| `FUNCTIONS_ENABLED` | bool | `false` | No | Enable the serverless Functions runtime. |
| `MAILPIT_ENABLED` | bool | `false` | No | Enable Mailpit (local email testing UI). |
| `MLFLOW_ENABLED` | bool | `false` | No | Enable MLflow (ML experiment tracking). |

> **Note:** Each optional service also exposes its own `*_VERSION`, `*_PORT`, `*_MEM_LIMIT`, and `*_CPU_LIMIT` vars. See the dedicated page for each service.

---

## Custom Services (CS\_N)

ɳSelf supports up to 10 user-defined services, numbered `CS_1` through `CS_10`. Each slot uses a consistent set of vars with `N` replaced by the slot number.

### Definition var

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `CS_N` | string | *(empty)* | No | Service definition string. Format: `name:template[:port][:route]`. Example: `ping_api:node:8001:ping`. |

### Per-service vars

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `CS_N_PORT` | int | *(from definition)* | No | Port the custom service listens on. |
| `CS_N_NAME` | string | *(from definition)* | No | Service name used in container labels and Nginx routing. |
| `CS_N_MEMORY` | string | `256m` | No | Docker memory limit. |
| `CS_N_CPU` | string | `0.5` | No | Docker CPU core limit. |
| `CS_N_PUBLIC` | bool | `false` | No | When `true`, the service is reachable from outside the Docker network via Nginx. |
| `CS_N_REPLICAS` | int | `1` | No | Number of container instances to run. |

**Example** (from `web/`, `nself.org` infrastructure):

```
CS_1=ping_api:node:8001:ping
CS_1_PORT=8001
CS_1_NAME=ping_api
CS_1_MEMORY=128m
CS_1_CPU=0.25
CS_1_PUBLIC=true
CS_1_REPLICAS=1
```

This registers a Node.js service named `ping_api` accessible at `ping.{BASE_DOMAIN}`.

---

## Remote Deploy

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `NSELF_DEPLOY_KEY_PATH` | string | *(empty)* | No | Path to the SSH private key used for remote deploys. Overrides `NSELF_DEPLOY_SSH_KEY` when set. Example: `~/.ssh/nself_deploy_ed25519`. |

---

## Embedded PostgreSQL (pglite/wasmtime)

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `NSELF_EMBEDDED_PG` | bool | `false` | No | When `true`, uses embedded PostgreSQL via pglite/wasmtime instead of a Docker Postgres container. Hasura connects via a Unix-domain socket bridge. Requires a `CGO_ENABLED=1` build of the CLI. Pass `--embedded-pg` to `nself start` as well. |

---

## Computed Variables

The following variables are derived automatically and written to `.env.computed` on every `nself build` and `nself start`. Do not set these manually, they will be overwritten.

| Variable | Derived From | Description |
|---|---|---|
| `DATABASE_URL` | `POSTGRES_*` vars | Full PostgreSQL connection string. |
| `DOCKER_NETWORK` | `PROJECT_NAME` | Docker network name: `nself_{PROJECT_NAME}`. |
| `AUTH_SMTP_SENDER` | `BASE_DOMAIN` | Defaults to `noreply@{BASE_DOMAIN}` if not explicitly set. |

---

← [[Configuration]] | [[Config-Postgres]] →
