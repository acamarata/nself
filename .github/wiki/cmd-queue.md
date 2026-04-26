# nself queue

> Manage async job queues (pg-boss).

## Synopsis

```
nself queue <subcommand> [flags] [args]
```

## Description

`nself queue` inspects and manages the async job system that runs on top of `pg-boss` in your project's Postgres. Use it to list queues with pending/active/failed/dead counts, drill into individual jobs, retry failed jobs, purge old jobs by age, and list scheduled cron jobs.

`queue list` summarizes every queue. `queue jobs <queue>` paginates jobs in a single queue, optionally filtered by `--state` (pending, active, failed, dead). `queue retry <job-id>` re-enqueues a job. `queue purge <queue>` deletes completed/dead jobs older than `--older-than` (default 30 days) to keep the queue table lean.

`queue cron list` enumerates registered cron schedules so operators can confirm that scheduled tasks (digests, backups, exports, license refreshes) are wired correctly.

## Subcommands

| Name | Description |
|------|-------------|
| `list` | List all queues with job counts |
| `jobs <queue>` | List jobs in a queue |
| `retry <job-id>` | Retry a failed or dead job |
| `purge <queue>` | Purge completed/dead jobs older than specified duration |
| `cron list` | List scheduled cron jobs |

## Flags

### `queue list`

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | JSON output |

### `queue jobs <queue>`

| Flag | Default | Description |
|------|---------|-------------|
| `--state` | `""` | Filter by state (`pending`, `active`, `failed`, `dead`) |
| `--limit` | `50` | Maximum jobs to return |
| `--json` | false | JSON output |

### `queue purge <queue>`

| Flag | Default | Description |
|------|---------|-------------|
| `--older-than` | `720h` | Purge jobs older than duration (default 30d) |

### `queue cron list`

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | JSON output |

## Examples

```bash
# Summarize every queue
nself queue list

# Drill into the email queue, only failed jobs, top 100
nself queue jobs email --state failed --limit 100

# Retry a single failed job
nself queue retry 0e9b5d9d-3e9e-4b1f-9b0e-ea25c3f9a8e1

# Purge dead jobs older than 7 days from the export queue
nself queue purge export --older-than 168h

# List all scheduled cron jobs
nself queue cron list
```

## See Also

- [[cmd-webhooks]], webhook outbox
- [[cmd-watchdog]], self-healing watchdog
- [[cmd-status]], service health
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
