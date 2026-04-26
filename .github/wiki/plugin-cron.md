# Cron Plugin

> Cron job scheduler with HTTP callbacks and Postgres-backed job queue. **Free — MIT licensed.**

## Install

```bash
nself plugin install cron
```

Redis is auto-enabled when the cron plugin is installed (BullMQ dependency). If `REDIS_ENABLED` is unset in your env, `nself build` detects the installed plugin and adds the redis service automatically.

## What It Does

Schedules recurring tasks using standard cron syntax. Jobs fire HTTP POST callbacks to any endpoint in your stack. Job history, status, and failures are stored in Postgres — no external scheduler or systemd timers needed.

Two ways to declare jobs:

- **Env-driven bootstrap** — declare jobs as `CRON_JOB_<N>_*` env vars. Upserted into Postgres on container start. Survives `nself rebuild`. Recommended for operator-owned infra tasks.
- **API** — create jobs via REST (`POST /v1/jobs`). Stored in Postgres, persisted across restarts. Recommended for application-created jobs.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CRON_PORT` | `3051` | Cron service port |
| `CRON_INTERNAL_SECRET` | *(auto-generated)* | Internal auth secret |
| `CRON_TIMEOUT_SECS` | `30` | HTTP callback timeout in seconds |
| `CRON_RETENTION_DAYS` | `90` | Run history retention in days |

## Env-Driven Schedule Bootstrap

Declare jobs as infrastructure-as-code in your `.env` file. The cron container upserts them into Postgres on every start — schedules survive `nself rebuild` without API calls.

```bash
# Format
CRON_JOB_<N>_SCHEDULE=<cron-expression>   # required
CRON_JOB_<N>_COMMAND=<callback-url>       # required — receives POST on each tick
CRON_JOB_<N>_NAME=<display-name>          # optional (default: "env-job-<N>")
CRON_JOB_<N>_PAYLOAD=<json-body>          # optional — sent as POST body
```

N ranges 1 to 20. Gaps are fine. If a job with the same name already exists, its schedule and callback URL are updated in place.

### Sample Configs

**Nightly Postgres backup to MinIO/S3 (3 AM UTC):**

```bash
CRON_JOB_1_SCHEDULE=0 3 * * *
CRON_JOB_1_COMMAND=http://backup:8080/backup/run
CRON_JOB_1_NAME=nightly-pg-backup
```

**ICS feed regen — prayer times, events (midnight UTC):**

```bash
CRON_JOB_2_SCHEDULE=0 0 * * *
CRON_JOB_2_COMMAND=http://my-service:8080/tasks/regen-ics
CRON_JOB_2_NAME=ics-regen
```

**Weekly email digest via Elastic Email (Monday 8 AM UTC):**

```bash
CRON_JOB_3_SCHEDULE=0 8 * * 1
CRON_JOB_3_COMMAND=http://my-service:8080/tasks/email-digest
CRON_JOB_3_NAME=weekly-email-digest
CRON_JOB_3_PAYLOAD={"list":"subscribers","template":"weekly"}
```

**Hasura health probe (every 5 minutes):**

```bash
CRON_JOB_4_SCHEDULE=*/5 * * * *
CRON_JOB_4_COMMAND=http://hasura:8080/healthz
CRON_JOB_4_NAME=hasura-health-probe
```

## Rebuild Safety

Jobs declared via `CRON_JOB_<N>_*` env vars are stored in Postgres (`np_cron_jobs`). Because Postgres data persists in a named Docker volume (`postgres_data`), jobs survive:

- `nself rebuild` — regenerates `docker-compose.yml`; the cron container restarts with the same env vars and re-seeds the same jobs (upsert — no duplicates)
- Server restart — Postgres volume is preserved; jobs reload from DB immediately
- Plugin reinstall — migration is idempotent; existing data is untouched

Jobs created via the REST API also persist in Postgres and survive rebuilds.

## nSelf-First Guard

The cron container is defined by `nself build` from the plugin's `docker-compose.plugin.yml`. Never add a cron service to a hand-written compose file — it will be overwritten on the next `nself build`.

```bash
# Correct
nself plugin install cron
nself build
nself start

# Never do this
# (editing docker-compose.yml directly)
```

## Auto-Enable: Redis

The cron plugin requires Redis (BullMQ queue for job overlap detection and retry). When `REDIS_ENABLED` is unset, `nself build` auto-enables Redis when it detects the cron plugin is installed:

```
Note: Redis auto-enabled because cron/notify plugin detected.
```

You can also declare `REDIS_ENABLED=true` explicitly in `.env` — the result is the same.

## Ports

| Port | Purpose |
|------|---------|
| 3051 | Cron service REST API (internal only) |

## Database Tables

Two tables added to your Postgres database:

- `np_cron_jobs` — job definitions and schedules
- `np_cron_runs` — execution history and status

The `np_cron_jobs.name` column has a unique index — used as the natural key for env-driven upserts.

## API

All endpoints are internal (not exposed via nginx). Call them from within your stack.

```
GET  /health              — Health check
GET  /v1/jobs             — List all jobs
POST /v1/jobs             — Create a job
GET  /v1/jobs/{id}        — Get job + run history
PUT  /v1/jobs/{id}        — Update a job
DELETE /v1/jobs/{id}      — Delete a job
POST /v1/jobs/{id}/trigger — Trigger a job immediately
```

### Create a Job via API

```bash
curl -X POST http://plugin-cron:3051/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "content-pipeline",
    "cron_expr": "0 2 * * *",
    "callback_url": "http://my-service:8080/tasks/run-pipeline"
  }'
```

## Nginx Routes

None — cron service is internal only. Not exposed on the public domain.

## Related

- [[plugin-cron-pro]] — distributed locks, visual dashboard, failure alerts, Slack/email notifications
- [[plugin-backup]] — backup plugin that pairs well with a nightly cron job
- [[plugin-notify]] — notification plugin; both cron and notify auto-enable Redis together

[[Home]]
