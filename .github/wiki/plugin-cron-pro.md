# Cron Plugin (Pro)

> pg_cron-backed job scheduler with HMAC-signed webhook delivery, per-account quotas, distributed leader lock, DLQ, and retry backoff. **Pro plugin — ɳClaw bundle.**

## Install

```bash
nself license set <your-pro-license-key>
nself plugin install cron
```

Requires an active ɳClaw bundle license or ɳSelf+ subscription.

## What It Does

Schedules recurring jobs using standard cron syntax (plus descriptors like `@daily`). Each job fires an HTTP POST to a configured webhook endpoint, signed with HMAC-SHA256 so the receiver can verify authenticity. Run history, failure streaks, and per-account quotas are stored in Postgres.

Key capabilities:

- **Distributed leader lock** — one active scheduler across multiple replicas via `np_cron_leader` heartbeat table
- **Per-account isolation** — all tables scoped by `source_account_id`; multi-app deployments get independent job namespaces
- **HMAC-signed delivery** — per-job signing key; receivers validate `X-Nself-Signature` header
- **DLQ + auto-disable** — configurable failure streak threshold; jobs flip to `permanently_failed` and stop retrying
- **Retry with backoff** — up to 5 attempts per job dispatch, configurable per job
- **Timezone support** — per-job IANA timezone for schedule evaluation
- **Agent dispatch** — target an ɳClaw subagent instead of a webhook (`delivery: agent`)
- **Run history** — full execution log with status, duration, attempt count, idempotency key

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CRON_PORT` | `3713` | Cron service port |
| `CRON_INTERNAL_SECRET` | auto-generated | Internal auth secret |
| `CRON_WEBHOOK_SIGNING_SECRET` | auto-generated | HMAC signing secret |
| `CRON_MAX_RETRIES` | `3` | Maximum delivery attempts per job |
| `CRON_TIMEOUT_SECS` | `30` | Webhook call timeout in seconds |
| `CRON_RETENTION_DAYS` | `30` | Run history retention in days |
| `CRON_DEFAULT_QUOTA_PER_ACCOUNT` | `100` | Default max jobs per account |
| `CRON_FAILURE_ALERT_THRESHOLD` | `5` | Consecutive failures before alert |
| `CRON_POLL_INTERVAL_SECS` | `10` | Scheduler poll interval |
| `CRON_LEADER_ENABLED` | `true` | Enable distributed leader lock |
| `CRON_INSTANCE_ID` | hostname | This instance's leader identity |

## Port

| Port | Purpose |
|------|---------|
| `3713` | Cron REST API and health endpoint |

## Database Tables

Six tables added to your Postgres database (all scoped to your app's schema):

| Table | Purpose |
|-------|---------|
| `np_cron_jobs` | Job definitions with schedule, delivery, and state |
| `np_cron_history` | Per-run execution records with timings and results |
| `np_cron_account_quota` | Per-account max-jobs overrides |
| `np_cron_account_usage` | Cumulative execution cost accounting per account |
| `np_cron_leader` | Distributed leader lock (singleton row) |
| `np_cron_web_snapshots` | Web-watch content hash cache |

## Hasura GraphQL

All user-facing tables expose GraphQL operations filtered by `source_account_id`. Users can only read and write their own jobs.

| Table | user role | nself_admin role |
|-------|-----------|-----------------|
| `np_cron_jobs` | select/insert/update/delete (own jobs) | full CRUD |
| `np_cron_history` | select (own account) | full CRUD |
| `np_cron_account_quota` | select (own account) | full CRUD |
| `np_cron_account_usage` | select (own account) | full CRUD |
| `np_cron_leader` | none | full CRUD |
| `np_cron_web_snapshots` | none | full CRUD |

## API Routes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/cron/jobs` | List jobs for the authenticated account |
| `POST` | `/cron/jobs` | Create a new job |
| `GET` | `/cron/jobs/{id}` | Get job by ID |
| `PATCH` | `/cron/jobs/{id}` | Update job fields |
| `DELETE` | `/cron/jobs/{id}` | Delete a job |
| `POST` | `/cron/jobs/{id}/run` | Trigger a job immediately |
| `POST` | `/cron/jobs/{id}/revive` | Clear permanently_failed state |
| `GET` | `/cron/history` | Paginated run history |
| `GET` | `/cron/quota` | Account quota info |
| `PUT` | `/cron/quota` | Update quota (admin only) |
| `GET` | `/cron/usage` | Account usage stats |
| `GET` | `/cron/status` | Scheduler health |
| `GET` | `/cron/failing` | Jobs in failing/failed state |
| `GET` | `/cron/config` | Runtime configuration |
| `PUT` | `/cron/config` | Update runtime configuration |

## Docker Hub

The pre-built image is available at:

```bash
docker pull nself/plugin-cron:latest
```

Multi-arch: `linux/amd64` and `linux/arm64`.

## Implementation Details

| Field | Value |
|-------|-------|
| Language | Go |
| Port | 3713 |
| Bundle | ɳClaw |
| License | Source-Available |
| Min nself version | 1.0.0 |
| Docker image | `nself/plugin-cron:latest` |

---

[[Home]] | [[Plugin-Overview]] | [[plugin-cron]] (free tier)
