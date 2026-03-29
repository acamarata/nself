# Optional Services Configuration

nSelf ships with four required services (PostgreSQL, Hasura, Auth, Nginx) that always run. Everything else is optional — you enable only what your project needs. This page covers all seven optional services: their toggle variables, ports, and the additional configuration each one requires.

---

## Summary

| Service | Toggle Variable | Default | Internal Port(s) | Auto-enabled When |
|---------|----------------|---------|-----------------|-------------------|
| Redis | `REDIS_ENABLED` | `false` | `6379` | `ai`, `claw`, or `mux` plugins installed |
| MinIO (Storage) | `MINIO_ENABLED` | `false` | `9000` (API), `9001` (console) | — |
| Mail (Mailpit) | `MAILPIT_ENABLED` | `false` | `1025` (SMTP), `8025` (UI) | — |
| Search | `SEARCH_ENABLED` | `false` | `7700` (MeiliSearch) / `8108` (Typesense) | — |
| Functions | `FUNCTIONS_ENABLED` | `false` | `3008` | — |
| MLflow | `MLFLOW_ENABLED` | `false` | `5000` | — |
| Admin | `NSELF_ADMIN_ENABLED` | `false` | `3021` | — |

The canonical way to enable any optional service is:

```bash
nself service enable <name>
```

This writes the toggle variable to your `.env.dev` file and rebuilds the compose configuration. You can also set the variable manually in your env file and run `nself build` to regenerate.

---

## Redis

Redis provides caching, session storage, and queue backends. It is automatically enabled when the `ai`, `claw`, or `mux` plugins are installed, because all three require a Redis instance.

### Enable

```bash
nself service enable redis
```

### Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `REDIS_ENABLED` | `false` | No | Enable Redis. Set to `true` automatically when `ai`/`claw`/`mux` plugins are installed. |
| `REDIS_VERSION` | `7-alpine` | No | Docker image tag. |
| `REDIS_PORT` | `6379` | No | Port Redis listens on inside the container. |
| `REDIS_PASSWORD` | *(empty)* | No | Auth password. Must be at least 16 characters if set. Leave empty for development; always set in production. |
| `REDIS_MEMORY` | `512M` | No | Docker memory limit for the Redis container. |
| `REDIS_CPU` | `0.5` | No | CPU core limit for the Redis container. |

### Notes

- Redis is accessed internally by other containers at `redis:6379`.
- There is no Nginx route for Redis — it is never exposed to the internet directly.
- If you are not using any AI plugins but your application needs Redis directly, set `REDIS_ENABLED=true` in your `.env.dev`.
- In production, always set a strong `REDIS_PASSWORD` (16+ characters).

---

## MinIO (Storage)

MinIO provides S3-compatible object storage. Use it to store file uploads, static assets, and any binary data your application produces. The Hasura Storage service requires MinIO when file handling is needed.

### Enable

```bash
nself service enable minio
```

`MINIO_ENABLED` and `STORAGE_ENABLED` are aliases — either will enable the service.

### Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `MINIO_ENABLED` / `STORAGE_ENABLED` | `false` | No | Enable MinIO. Both variables are aliases for the same toggle. |
| `MINIO_PORT` | `9000` | No | S3-compatible API port. |
| `MINIO_CONSOLE_PORT` | `9001` | No | MinIO web console port. |
| `MINIO_ROOT_USER` | `minioadmin` | No | Root username. Change in production. |
| `MINIO_ROOT_PASSWORD` | `minioadmin` | **Yes (prod)** | Root password. Must be at least 16 characters in production. |
| `MINIO_DEFAULT_BUCKETS` | `uploads,public,private,temp` | No | Comma-separated list of buckets created automatically on first start. |
| `MINIO_REGION` | `us-east-1` | No | Region reported by the MinIO S3 API. |
| `S3_ACCESS_KEY` | `storage-access-key-dev` | No | S3 access key used by application services (Hasura Storage, Functions, etc.). |
| `S3_SECRET_KEY` | *(required if enabled)* | **Yes** | S3 secret key. Must be set if MinIO is enabled. |
| `S3_BUCKET` | `nself` | No | Default bucket name used by application services. |

### Accessing the Console

The MinIO console is available at `http://localhost:{MINIO_CONSOLE_PORT}` (default: `http://localhost:9001`) during development. Log in with `MINIO_ROOT_USER` and `MINIO_ROOT_PASSWORD`.

### Production Notes

- Change `MINIO_ROOT_USER` from the default `minioadmin`.
- Set `MINIO_ROOT_PASSWORD` and `S3_SECRET_KEY` to strong random values in `.env.secrets`.
- The console port (`9001`) is not exposed via Nginx by default. To expose it, configure an Nginx route explicitly.
- Run `nself config generate-secret` to produce strong random values for both password fields.

---

## Mail (Mailpit)

Mailpit is a development mail server that captures all outgoing emails and displays them in a web UI. It is intended for local development and staging environments only. In production, configure a real SMTP provider instead.

### Enable

```bash
nself service enable mail
```

### Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `MAILPIT_ENABLED` | `false` | No | Enable Mailpit. For development and staging only. |
| `MAILPIT_SMTP_PORT` | `1025` | No | SMTP port. Other services send email to this port. |
| `MAILPIT_UI_PORT` | `8025` | No | Web UI port for viewing captured messages. |
| `MAILPIT_MAX_MESSAGES` | `500` | No | Maximum number of messages to retain before the oldest are discarded. |
| `MAILPIT_ROUTE` / `MAIL_ROUTE` | `mail` | No | Nginx subdomain prefix. With `BASE_DOMAIN=example.com`, the UI is available at `mail.example.com`. |
| `MAILPIT_UI_USER` | `admin` | No | Basic auth username for the web UI (applied in staging/prod). |
| `MAILPIT_UI_PASSWORD` | *(auto-generated in staging/prod)* | No | Basic auth password for the web UI. |

### Production Email

Mailpit is a dev tool and should not be used in production. To send real email in production, set the Auth SMTP variables instead:

```bash
# .env.prod
AUTH_SMTP_HOST=smtp.sendgrid.net
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=apikey
AUTH_SMTP_PASS=your-sendgrid-api-key
AUTH_SMTP_SENDER=noreply@yourdomain.com
```

The `AUTH_SMTP_*` variables configure the Auth service to deliver email via your chosen provider. Any provider that supports standard SMTP works — SendGrid, Postmark, Mailgun, Resend, AWS SES, and others.

---

## Search

nSelf supports multiple search engines. The default is MeiliSearch, but you can switch to Typesense, Elasticsearch, OpenSearch, Zinc, or Sonic by changing `SEARCH_ENGINE`.

### Enable

```bash
nself service enable search
```

### Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `SEARCH_ENABLED` | `false` | No | Enable the search service. |
| `SEARCH_ENGINE` / `SEARCH_PROVIDER` | `meilisearch` | No | Search engine to use. Supported values: `meilisearch`, `typesense`, `elasticsearch`, `opensearch`, `zinc`, `sonic`. |
| `SEARCH_PORT` | `7700` | No | Service port. Automatically set based on the chosen engine — override only if you have a port conflict. |
| `SEARCH_API_KEY` | *(auto-generated)* | No | Primary API key. Generated automatically if not set. |
| `SEARCH_ROUTE` | `search` | No | Nginx subdomain prefix for the search API. |

### Engine-Specific Variables

#### MeiliSearch (default)

| Variable | Default | Description |
|----------|---------|-------------|
| `MEILISEARCH_MASTER_KEY` | `meilisearch-master-key-minimum-16-chars` | MeiliSearch master key. Change in production. |

MeiliSearch listens on port `7700` by default.

#### Typesense

| Variable | Default | Description |
|----------|---------|-------------|
| `TYPESENSE_API_KEY` | `typesense-api-key-minimum-32-chars` | Typesense API key. Change in production. |

Typesense listens on port `8108` by default.

### Switching Engines

To switch from MeiliSearch to Typesense:

```bash
# .env.dev
SEARCH_ENGINE=typesense
TYPESENSE_API_KEY=your-strong-key-here
```

Then run `nself build` to regenerate the compose configuration, followed by `nself restart search`.

---

## Functions (Serverless)

The Functions service provides a serverless runtime for custom business logic. It mounts the `./functions/` directory from your project and makes each file available as an HTTP endpoint via Nginx.

### Enable

```bash
nself service enable functions
```

### Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `FUNCTIONS_ENABLED` | `false` | No | Enable the serverless functions runtime. |
| `FUNCTIONS_VERSION` | `latest` | No | Docker image tag for the Functions container. |
| `FUNCTIONS_PORT` | `3008` | No | Port the Functions service listens on. |
| `FUNCTIONS_ROUTE` | `functions` | No | Nginx subdomain prefix. With `BASE_DOMAIN=example.com`, functions are available at `functions.example.com`. |

### Project Structure

When Functions is enabled, nSelf mounts `./functions/` from your project root into the container:

```
your-project/
  functions/
    hello.ts          → GET /hello
    users/
      create.ts       → POST /users/create
```

Each TypeScript or JavaScript file exports a default handler function. See the Functions documentation for the full request/response API.

---

## MLflow

MLflow is a machine learning experiment tracking platform. Enable it when your project involves model training, experiment comparison, or artifact storage. MLflow requires MinIO to be enabled when artifact storage is needed.

### Enable

```bash
nself service enable mlflow
```

### Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `MLFLOW_ENABLED` | `false` | No | Enable MLflow. |
| `MLFLOW_PORT` | `5000` | No | Port the MLflow tracking server listens on. |
| `MLFLOW_ROUTE` | `mlflow` | No | Nginx subdomain prefix. |
| `MLFLOW_DB_NAME` | `mlflow` | No | Name of the PostgreSQL database used by MLflow for experiment metadata. Created automatically. |
| `MLFLOW_ARTIFACTS_BUCKET` | `mlflow-artifacts` | No | MinIO bucket used to store model artifacts. Created automatically when MinIO is enabled. |
| `MLFLOW_AUTH_ENABLED` | `false` | No | Enable HTTP basic auth on the MLflow UI. Strongly recommended in staging and production. |
| `MLFLOW_AUTH_USERNAME` | `admin` | No | Basic auth username for the MLflow UI. |
| `MLFLOW_AUTH_PASSWORD` | *(required if auth enabled)* | **Yes (if auth enabled)** | Basic auth password. Must be set when `MLFLOW_AUTH_ENABLED=true`. |

### Dependency on MinIO

MLflow stores experiment artifacts (model files, plots, datasets) in object storage. When `MLFLOW_ENABLED=true` and `MINIO_ENABLED=true`, nSelf automatically creates the `MLFLOW_ARTIFACTS_BUCKET` bucket in MinIO and configures MLflow to use it.

If MinIO is not enabled, MLflow falls back to local filesystem artifact storage, which is only appropriate for single-node development use.

### Production Notes

- Set `MLFLOW_AUTH_ENABLED=true` in staging and production to protect the tracking UI.
- Set a strong `MLFLOW_AUTH_PASSWORD` in `.env.secrets`.
- Enable MinIO alongside MLflow for persistent artifact storage.

---

## Admin Dashboard

The nSelf Admin dashboard is a local GUI companion for your nSelf stack. It runs at `localhost:3021` on your own machine. It is never deployed to a server and is never accessible from the internet.

### Enable

```bash
nself service enable admin
```

Alternatively, use the dedicated command:

```bash
nself admin start
```

### Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `NSELF_ADMIN_ENABLED` | `false` | No | Enable the Admin dashboard container. |
| `NSELF_ADMIN_PORT` | `3021` | No | Port the dashboard listens on. Access it at `http://localhost:3021`. |
| `NSELF_ADMIN_VERSION` | `latest` | No | Docker image tag for `nself/nself-admin`. |
| `NSELF_ADMIN_USER` | `admin` | No | Login username for the dashboard. |
| `NSELF_ADMIN_PASSWORD` | *(required in prod)* | **Yes (prod)** | Login password. Must be at least 16 characters in production. |

### Important Constraints

- Admin is **localhost-only**. It binds to `127.0.0.1` and is not reachable from outside your machine.
- Admin is **not a hosted service**. nSelf does not run Admin on any server. Each developer runs it locally alongside their stack.
- Admin is distributed as the `nself/nself-admin` Docker image and is pulled automatically when enabled.
- Do not add an Nginx route for Admin — it is intentionally isolated to your local machine.

---

## Enabling Multiple Services at Once

You can enable several services in a single command:

```bash
nself service enable redis minio functions
```

Or set the toggle variables in your env file and run `nself build`:

```bash
# .env.dev
REDIS_ENABLED=true
MINIO_ENABLED=true
FUNCTIONS_ENABLED=true
```

```bash
nself build && nself restart
```

---

## Checking Service Status

To see which optional services are currently running:

```bash
nself status
```

To get service URLs including optional service endpoints:

```bash
nself urls
```

---

← [[Config-Nginx]] | [[Configuration]] | [[Config-Custom-Services]] →
