# plugin-cron (paid) — ɳClaw Cron Scheduler

## Overview

The paid cron plugin (`port 3713`) provides enterprise-grade scheduled job execution for ɳClaw subscribers. It extends the free core cron plugin with advanced features: AI-driven cron definitions via natural language, webhook delivery with HMAC signing, per-account quotas, timezone-aware scheduling, and full audit history.

## Bundle

- **Bundle**: ɳClaw
- **Port**: 3713
- **License**: Requires active ɳClaw or ɳSelf+ license
- **Category**: Automation

## Installation

```bash
nself plugin install cron
```

License is validated automatically at startup. Set `NSELF_LICENSE_KEY` in your environment or via `nself config set license-key <key>`.

## Features

- **AI-defined crons** — describe a schedule in plain language; ɳClaw converts it to cron syntax
- **Webhook delivery** — signed `X-Nself-Signature` header for verification
- **Per-account quotas** — configurable max jobs and execution rate
- **Timezone support** — any IANA timezone, e.g., `America/New_York`
- **Retry policies** — exponential backoff with configurable max attempts
- **Web snapshot jobs** — capture URL screenshots on schedule
- **Full audit history** — every execution logged in `np_cron_history`

## Database Tables

All tables use the `np_cron_` prefix and include `source_account_id` for multi-app isolation.

| Table | Purpose |
|-------|---------|
| `np_cron_jobs` | Scheduled job definitions |
| `np_cron_history` | Execution history and results |
| `np_cron_account_quota` | Per-account rate limits |
| `np_cron_account_usage` | Running usage counters |
| `np_cron_web_snapshots` | Web page snapshot jobs |
| `np_cron_leader` | Leader election for HA deployments |

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `NSELF_LICENSE_KEY` | Yes | ɳClaw or ɳSelf+ license key |
| `CRON_MAX_JOBS_PER_ACCOUNT` | No | Default: 100 |
| `CRON_WEBHOOK_TIMEOUT_SECONDS` | No | Default: 30 |
| `CRON_MAX_RETRIES` | No | Default: 3 |
| `PORT` | No | Default: 3713 |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Plugin health check |
| `GET` | `/jobs` | List all scheduled jobs |
| `POST` | `/jobs` | Create a new job |
| `GET` | `/jobs/:id` | Get job details |
| `PUT` | `/jobs/:id` | Update a job |
| `DELETE` | `/jobs/:id` | Delete a job |
| `GET` | `/jobs/:id/history` | Execution history for a job |
| `POST` | `/jobs/:id/run` | Trigger immediate execution |
| `GET` | `/quota` | Account quota usage |

## Example: Create a Job

```json
POST /jobs
{
  "name": "Daily report",
  "schedule": "0 9 * * 1-5",
  "timezone": "America/New_York",
  "delivery": "webhook",
  "webhook_url": "https://example.com/hooks/daily-report",
  "payload": {"report_type": "summary"},
  "max_attempts": 3,
  "sign_payload": true
}
```

## Security

- Webhook signatures use HMAC-SHA256 with a per-job secret
- All rows are isolated by `source_account_id` (Hasura RLS enforced)
- License validated at every request — expired or invalid keys are rejected with `HTTP 402`

## Hasura Integration

All `np_cron_*` tables are tracked in Hasura with row-level security:

```yaml
filter:
  source_account_id:
    _eq: X-Hasura-Source-Account-Id
```

## Changelog

- **1.0.0** — Initial release (ɳClaw bundle, port 3713)
- **1.1.0** — AI cron definitions, web snapshot jobs
- **1.2.0** — Agent-aware cron (agent_id, agent_context, agent_tools columns)
