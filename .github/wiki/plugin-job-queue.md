# Job Queue Plugin

> Redis-backed persistent job queue with at-least-once delivery, priority queuing, exponential backoff retry, and a dead-letter queue (DLQ) Admin UI panel. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|----------------------|
| Free | $0 | $0 | No |
| Any Bundle | $0.99/mo | $9.99/yr | Yes |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** Any paid bundle or ɳSelf+ (infrastructure plugin, available to all licensed tiers).

## Bundle membership

This is an infrastructure plugin available across all bundles. It is not locked to a specific bundle:

- **All bundles** ($0.99/mo or $9.99/yr) — ɳClaw, ɳChat, ɳTV, ɳFamily, ClawDE, ɳSentry
- **ɳSelf+** ($3.99/mo or $39.99/yr) — all bundles + all apps + priority support

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install job-queue
nself build
```

The license is validated against `ping.nself.org/license/validate`. If the key lacks the required entitlement, install will fail with a clear error message.

## Description

The job-queue plugin adds durable background job processing to any nSelf deployment. It connects Redis (for execution) and Postgres (for visibility and durability), runs configurable worker pools, and exposes an HTTP API for enqueuing and monitoring jobs.

Workers use `BRPOPLPUSH` for atomic at-least-once delivery. If a worker crashes mid-job, the job stays in the processing set and is picked up on the next service start. Each job is retried with exponential backoff (2^attempt seconds, capped at one hour) up to `JOBQUEUE_MAX_ATTEMPTS` times. Once all attempts are exhausted, the job moves to the dead-letter queue for operator review.

The Admin UI includes a DLQ panel where operators can see failed jobs with their error messages, requeue a job (reset to pending with cleared attempt count), or permanently discard a job (mark as dead and remove from DLQ).

Priority queuing is built in. High-priority jobs are placed at the right end of the Redis list, where `BRPOPLPUSH` consumes first. Medium and low jobs go to the left end and are processed after the high-priority queue drains.

## CS_10 Custom Service Slot

This plugin occupies the **CS_10** custom service slot. Custom service slots are reserved port ranges and Compose service names in the nSelf runtime, set aside for plugins that run as standalone services outside the nSelf core microservices.

| Slot | Plugin | Port |
|------|--------|------|
| CS_10 | job-queue | 8213 |

The CS_10 slot is assigned exclusively to job-queue. Changing the slot requires updating `cs_slot` in `plugin.yaml` and the `JOBQUEUE_PORT` env var. Available slots are CS_1 through CS_20 (see SPORT `F08-SERVICE-INVENTORY.md`).

The service is accessible within your nSelf network at `http://localhost:8213`. It is not exposed publicly via nginx by default.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `JOBQUEUE_PORT` | `8213` | HTTP API port (CS_10 slot) |
| `JOBQUEUE_CONCURRENCY` | `5` | Worker goroutines per queue |
| `JOBQUEUE_MAX_ATTEMPTS` | `8` | Delivery attempts before DLQ routing |
| `JOBQUEUE_QUEUES` | `default,email,ai,media` | Comma-separated queue names |
| `REDIS_URL` | `redis://127.0.0.1:6379` | Redis connection URL |
| `DATABASE_URL` | — | Postgres connection string (required for DLQ and visibility) |

## Ports

| Port | Purpose |
|------|---------|
| 8213 | Job Queue HTTP API (CS_10 slot) |

## Database Schema

| Table | Purpose |
|-------|---------|
| `np_jobs` | Visibility snapshot of Redis job state (status, attempts, timestamps, error) |
| `np_job_dlq` | Jobs that exhausted all retry attempts, pending operator action |
| `np_circuit_breakers` | Persistent circuit breaker state for registered circuit names |

All three tables include `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation per the nSelf Multi-Tenant Convention Wall.

## REST API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Returns `{"status":"ok"}` when Redis is reachable |
| `POST` | `/jobs/enqueue` | Enqueue a background job |
| `GET` | `/jobs` | Queue depths per queue name |
| `GET` | `/dlq` | List DLQ entries (most recent first, limit 100) |
| `POST` | `/dlq/requeue` | Reset a DLQ job to pending for re-delivery |
| `POST` | `/dlq/discard` | Permanently discard a DLQ job (idempotent) |
| `GET` | `/metrics` | Prometheus queue depth gauges |

## Examples

### Enqueue a background job

```bash
curl -X POST http://localhost:8213/jobs/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "queue": "email",
    "job_type": "send_welcome",
    "payload": {"user_id": "u_abc123"},
    "priority": "high"
  }'
```

Response:
```json
{"id":"17f3a9b2...","status":"enqueued","priority":"high"}
```

### Check queue depths

```bash
curl http://localhost:8213/jobs
```

Response:
```json
{"ai":0,"default":3,"email":12,"media":1}
```

### View DLQ

```bash
curl http://localhost:8213/dlq
```

Response:
```json
[
  {
    "id": "job-uuid",
    "queue": "email",
    "job_type": "send_welcome",
    "final_error": "SMTP connection refused",
    "dlq_at": "2026-06-24T10:30:00Z"
  }
]
```

### Requeue a failed job

```bash
curl -X POST http://localhost:8213/dlq/requeue \
  -H "Content-Type: application/json" \
  -d '{"job_id":"job-uuid"}'
```

### Discard a failed job

```bash
curl -X POST http://localhost:8213/dlq/discard \
  -H "Content-Type: application/json" \
  -d '{"job_id":"job-uuid"}'
```

## Source

Source-available (license required to run): [`plugins-pro/paid/job-queue/`](https://github.com/nself-org/plugins-pro/tree/main/paid/job-queue)

Note: `plugins-pro` is a private repository. Source access is granted to ɳSelf+ subscribers and Enterprise customers.

## See Also

- [[plugin-cron]] — scheduled job execution (complementary to job-queue)
- [[plugin-notify]] — push notifications, useful for job completion events
- [[Pricing]] — tier comparison
- [[Plugins]] — full plugin index

← [[Plugins]] | [[Home]] →
