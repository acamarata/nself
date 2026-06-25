# Plugin: Job Queue

**Port:** 8213
**Bundle:** Unbundled — standalone install
**Custom Service Slot:** CS_10
**License:** ɳSelf+ required (`NSELF_JOB_QUEUE=true`)

The job-queue plugin provides durable background job queuing with Redis as the execution store and Postgres as the visibility/durability layer. It supports named queues, configurable concurrency, exponential-backoff retry, and a Dead Letter Queue (DLQ) for failed jobs.

## Installation

```bash
nself plugin install job-queue
```

## Job Schema

Jobs have the following fields in `np_jobs`:

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Unique job identifier |
| `queue` | TEXT | Queue name (e.g. `email`, `ai`) |
| `job_type` | TEXT | Handler identifier |
| `payload` | JSONB | Job data |
| `status` | TEXT | `pending`, `active`, `completed`, `failed`, `dead` |
| `attempt_count` | SMALLINT | Number of execution attempts |
| `scheduled_at` | TIMESTAMPTZ | When job becomes eligible |
| `source_account_id` | TEXT | Multi-app isolation key |

## Enqueueing Jobs

```bash
curl -X POST http://localhost:8213/jobs/enqueue \
  -H "Content-Type: application/json" \
  -d '{"queue":"email","job_type":"send_welcome_email","payload":{"user_id":"abc123"}}'
# → {"id":"18e4a1b2...","status":"enqueued"}
```

## Priority Values

Priority is implemented via dedicated queues. Route high-priority work to a queue with higher concurrency:

| Queue | Use |
|---|---|
| `default` | General background work |
| `email` | Email delivery |
| `ai` | AI/LLM inference jobs |
| `media` | Video/image processing |

Configure additional queues via `JOBQUEUE_QUEUES=default,email,ai,media,my_queue`.

## Retry Configuration

Failed jobs retry with exponential backoff: `2^attempt` seconds, capped at 1 hour.

| Variable | Default | Description |
|---|---|---|
| `JOBQUEUE_MAX_ATTEMPTS` | `8` | Max retries before DLQ |
| `JOBQUEUE_CONCURRENCY` | `5` | Worker goroutines per queue |

## Dead Letter Queue (DLQ)

Jobs exhausting all retry attempts land in the DLQ (`np_job_dlq`).

```bash
# List failed jobs
curl http://localhost:8213/dlq

# Requeue a failed job (resets attempt_count=0, status=pending)
curl -X POST http://localhost:8213/dlq/requeue \
  -d '{"job_id":"uuid-here"}'

# Permanently discard a job
curl -X POST http://localhost:8213/dlq/discard \
  -d '{"job_id":"uuid-here"}'
```

The Admin UI DLQ panel at `/admin/job-queue` displays failed jobs with their error messages and exposes requeue/discard actions with confirmation dialogs.

## CS_10 Custom Service Slot

This plugin occupies **custom service slot CS_10** in `F08-SERVICE-INVENTORY.md`. It reserves:

- Port `8213` for the HTTP API
- Redis list keys `queue:<name>` and `queue:<name>:processing` per configured queue
- Postgres tables `np_jobs`, `np_job_dlq`, `np_circuit_breakers`

Do not assign other services to CS_10 or port 8213.

## Handler Registration

Register job handlers in your application via the nSelf SDK:

```go
client.RegisterJobHandler("send_welcome_email", func(ctx context.Context, payload json.RawMessage) error {
    var data struct { UserID string `json:"user_id"` }
    json.Unmarshal(payload, &data)
    return sendWelcomeEmail(ctx, data.UserID)
})
```

Unregistered job types fail immediately and route through retry/DLQ. This prevents silent data loss.

## Metrics

Prometheus-format queue depth metrics at `/metrics`:

```
nself_jobqueue_queue_depth{queue="email"} 4
nself_jobqueue_queue_depth{queue="ai"} 12
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `NSELF_JOB_QUEUE` | — | `true` to start service (license gate) |
| `REDIS_URL` | `redis://127.0.0.1:6379` | Redis connection |
| `DATABASE_URL` | — | Postgres for visibility snapshots |
| `JOBQUEUE_PORT` | `8213` | HTTP listen port |
| `JOBQUEUE_CONCURRENCY` | `5` | Workers per queue |
| `JOBQUEUE_MAX_ATTEMPTS` | `8` | Max retries before DLQ |
| `JOBQUEUE_QUEUES` | `default,email,ai,media` | Comma-separated queue names |
