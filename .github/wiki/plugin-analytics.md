# Analytics Plugin

> Product analytics, event tracking, funnels, retention, and quota management. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install analytics
```

## What It Does

Tracks custom product events (page views, feature usage, conversions) and stores them in Postgres for querying via Hasura GraphQL. Provides funnel analysis, retention cohort reporting, and usage quota tracking per user or tenant. Designed for product analytics and internal dashboards, not marketing attribution.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ANALYTICS_PORT` | `3206` | Analytics service port |
| `ANALYTICS_RETENTION_DAYS` | `365` | Days to retain event data |
| `ANALYTICS_BATCH_SIZE` | `100` | Event ingestion batch size |
| `ANALYTICS_QUOTA_ENABLED` | `false` | Enable usage quota enforcement |

## Ports

| Port | Purpose |
|------|---------|
| 3206 | Analytics REST API |

## Database Tables

6 tables added to your Postgres database:
- `np_analytics_events`, raw event records
- `np_analytics_sessions`, user sessions
- `np_analytics_funnels`, funnel definitions
- `np_analytics_cohorts`, retention cohort data
- `np_analytics_quotas`, usage quota configurations
- `np_analytics_quota_usage`, quota consumption records

## Nginx Routes

| Route | Target |
|-------|--------|
| `/analytics/` | Analytics ingest and query API |

## API

```
POST /events          — Ingest events (single or batch)
GET  /funnels/{id}    — Funnel conversion report
GET  /retention       — Retention cohort report
GET  /quotas/{user}   — Current quota usage
```
