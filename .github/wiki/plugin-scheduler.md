# Scheduler Plugin

> Advanced job scheduler — cron + delay + priority + workflow with visual timeline UI. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install scheduler
```

## What It Does

A comprehensive job scheduling system that goes beyond standard cron. Supports one-time delayed jobs, recurring cron jobs, priority queues, job chains (run B after A succeeds), and dependency graphs. Includes a visual timeline UI for monitoring upcoming and running jobs. Integrates with the `jobs` plugin for execution.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SCHEDULER_PORT` | `3058` | Scheduler service port |
| `SCHEDULER_MAX_CONCURRENT` | `20` | Max concurrent job executions |
| `SCHEDULER_DEFAULT_TIMEZONE` | `UTC` | Default timezone for cron jobs |
| `SCHEDULER_HISTORY_DAYS` | `90` | Days to retain execution history |

## Ports

| Port | Purpose |
|------|---------|
| 3058 | Scheduler REST API and UI |

## Database Tables

5 tables added to your Postgres database:
- `np_scheduler_jobs` — job definitions
- `np_scheduler_schedules` — cron and delay schedules
- `np_scheduler_executions` — execution history
- `np_scheduler_chains` — job dependency chains
- `np_scheduler_queues` — priority queue configurations

## Nginx Routes

| Route | Target |
|-------|--------|
| `/scheduler/` | Scheduler API |
| `/scheduler/ui` | Visual timeline dashboard |
