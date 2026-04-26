# Jobs Plugin

> Background job queue with BullMQ-style dispatch and a built-in dashboard. **Free, MIT licensed.**

## Install

```bash
nself plugin install jobs
```

## What It Does

Provides a Redis-backed job queue for background processing. Supports priority queues, delayed jobs, retries with exponential backoff, and job deduplication. Includes a web dashboard for monitoring queue depth and job status.

## Dependencies

Requires Redis (`REDIS_ENABLED=true` in your `.env`).

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `JOBS_PORT` | `3105` | Jobs service port |
| `JOBS_CONCURRENCY` | `5` | Workers per queue |
| `JOBS_MAX_RETRIES` | `3` | Max retry attempts |
| `JOBS_RETRY_DELAY` | `60` | Seconds between retries |

## Ports

| Port | Purpose |
|------|---------|
| 3105 | Jobs service REST API and dashboard |

## Database Tables

2 tables added to your Postgres database:
- `np_jobs_definitions`, registered job types
- `np_jobs_history`, completed/failed job records

## Nginx Routes

| Route | Target |
|-------|--------|
| `/jobs-dashboard/` | Job queue dashboard UI |

## API

```
GET  /health         — Health check
POST /jobs           — Enqueue a job
GET  /jobs/{id}      — Get job status
DELETE /jobs/{id}    — Cancel a job
GET  /queues         — List queues and depths
```
