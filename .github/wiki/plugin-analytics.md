# Analytics Plugin

> Event tracking, counters, funnels, and quota management analytics engine. **Pro plugin.**

> **Requires:** a pro license tier. Set a key with `nself license set` before installing.

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install analytics
```

## What It Does

Tracks custom product events and increments named counters, then stores them in Postgres for querying via the plugin API or Hasura GraphQL. It computes funnel conversion across ordered steps, rolls counters up on a schedule, and enforces usage quotas per key with violation tracking. Designed for product analytics and internal usage metering, not marketing attribution.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ANALYTICS_PLUGIN_PORT` | `3206` | Analytics service port |
| `ANALYTICS_BATCH_SIZE` | `100` | Event ingestion batch size |
| `ANALYTICS_API_KEY` | (unset) | Bearer key required on API routes |
| `ANALYTICS_EVENT_RETENTION_DAYS` | `90` | Days to retain raw events |
| `ANALYTICS_COUNTER_RETENTION_DAYS` | `365` | Days to retain counter history |
| `ANALYTICS_ROLLUP_INTERVAL_MS` | `3600000` | Counter rollup interval |
| `ANALYTICS_RATE_LIMIT_MAX` | (unset) | Max requests per window |
| `ANALYTICS_RATE_LIMIT_WINDOW_MS` | (unset) | Rate limit window in milliseconds |

## Ports

| Port | Purpose |
|------|---------|
| 3206 | Analytics REST API |

## Database Tables

6 tables added to your Postgres database (no `np_` prefix; analytics plugin uses unprefixed names):
- `analytics_events`, raw event records
- `analytics_counters`, named counter values
- `analytics_funnels`, funnel definitions
- `analytics_quotas`, usage quota configurations
- `analytics_quota_violations`, recorded quota breaches
- `analytics_webhook_events`, inbound webhook event log

All tables carry `source_account_id` for multi-app isolation.

## API

All routes require a bearer token (`ANALYTICS_API_KEY`).

```
POST /api/v1/events              Ingest a single event
POST /api/v1/events/batch        Ingest a batch of events
GET  /api/v1/events              Query events
POST /api/v1/counters/increment  Increment a counter
GET  /api/v1/counters            List all counters
GET  /api/v1/counters/{key}      Read a counter value
GET  /api/v1/funnels             List funnels
POST /api/v1/funnels             Create a funnel
GET  /api/v1/funnels/{id}/analyze  Funnel conversion report
GET  /api/v1/quotas              List quotas
POST /api/v1/quotas              Create a quota
GET  /api/v1/violations          List quota violations
GET  /api/v1/stats               Summary stats
GET  /api/v1/stats/timeseries    Time-series stats
GET  /health                     Liveness probe
GET  /ready                      Readiness probe
```

## Related

[[plugin-overview|Plugin Overview]] · [[Home]]
