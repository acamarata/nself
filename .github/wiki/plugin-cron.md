# Cron Plugin

> Cron job scheduler with HTTP callbacks and Postgres-backed job queue. **Free — MIT licensed.**

## Install

```bash
nself plugin install cron
```

## What It Does

Schedules recurring tasks using standard cron syntax. Jobs fire HTTP callbacks to any endpoint in your stack. Job history, status, and failures are stored in Postgres. For distributed locks, a visual dashboard, and failure alerts, see [plugin-cron-pro](plugin-cron-pro).

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CRON_PORT` | `3051` | Cron service port |
| `CRON_SECRET` | *(auto-generated)* | Internal auth secret |
| `CRON_MAX_RETRIES` | `3` | Max retries on callback failure |

## Ports

| Port | Purpose |
|------|---------|
| 3051 | Cron service REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_cron_jobs` — job definitions and schedules
- `np_cron_runs` — execution history and status

## Nginx Routes

None — cron service is internal only.

## API

```
GET    /health      — Health check
GET    /jobs        — List all jobs
POST   /jobs        — Create a job
DELETE /jobs/{id}   — Delete a job
GET    /runs        — Job execution history
```
