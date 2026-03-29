> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# Cron Pro Plugin

> Advanced cron scheduler — distributed locks, complex syntax, dashboard, and failure alerts. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install cron-pro
```

## What It Does

Extends the free `cron` plugin with production-grade scheduling features: distributed locking to prevent duplicate runs across multiple nSelf instances, support for complex cron syntax including intervals and human-readable schedules, a visual dashboard for job monitoring, failure alerts via email and Slack, and a full job run history with timing analytics.

## Note on Free Tier

The free `cron` plugin handles standard cron syntax with HTTP callbacks. This pro version adds distributed locks, complex syntax, dashboard, and alerting.

## Implementation Details

- **Language:** Rust
- **Port:** 3720
- **Tables:** 2

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CRON_PRO_PORT` | `3720` | Cron pro service port |
| `CRON_PRO_LOCK_TTL` | `300` | Distributed lock TTL in seconds |
| `CRON_PRO_ALERT_EMAIL` | — | Email for failure alerts |
| `CRON_PRO_ALERT_SLACK` | — | Slack webhook URL for alerts |
| `CRON_PRO_HISTORY_DAYS` | `90` | Days to retain run history |

## Ports

| Port | Purpose |
|------|---------|
| 3720 | Cron pro REST API and dashboard |

## Database Tables

2 tables added to your Postgres database:
- `np_cron_pro_jobs` — job definitions with complex schedules
- `np_cron_pro_runs` — detailed execution history with timings

## Nginx Routes

| Route | Target |
|-------|--------|
| `/cron/dashboard` | Cron job monitoring dashboard |
