# plugin: audit-analytics

> Advanced audit log analytics for nSelf deployments. Anomaly detection, behaviour heatmaps, risk scoring, and privileged-action review queues. **ɳSelf+ license required.**

> **Requires:** ɳSelf+ license and `NSELF_AUDIT_ANALYTICS=true`.

## Overview

The `audit-analytics` plugin analyses the `np_audit_log` table (populated by the `audit` plugin) and surfaces security insights:

- **Statistical anomaly detection** — z-score over rolling 24h window per user
- **User behaviour heatmaps** — hourly action frequency by user and action type
- **Risk scoring** — composite score per session (deviation + privilege + time-of-day)
- **Privileged action review queue** — human review SLA for admin/destructive actions
- **Real-time alerts** — webhook + email on high-risk events (Enterprise)

## Port

**3714**

> **Port conflict warning:** Port 3714 is also registered for the `voice` plugin (ɳClaw bundle). Do not run both plugins on the same host using the default port. Set `AUDIT_ANALYTICS_PORT` to an alternate value if co-deploying. This conflict is tracked for resolution in the F10 port registry.

## Installation

```bash
nself plugin install audit-analytics
```

Set the license gate before starting:

```bash
export NSELF_AUDIT_ANALYTICS=true
export DATABASE_URL=postgres://...
nself plugin start audit-analytics
```

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | Postgres connection string |
| `NSELF_AUDIT_ANALYTICS` | Yes | Must be `true` (ɳSelf+ gate) |
| `AUDIT_ANALYTICS_PORT` | No | HTTP port (default `3714`) |
| `AUDIT_ANALYTICS_SHARED_SECRET` | No | Bearer token for auth (open-dev if unset) |
| `NSELF_AUDIT_ANOMALY_ZSCORE` | No | Z-score threshold (default `3.0`) |
| `NSELF_AUDIT_ANALYTICS_REALTIME` | No | Real-time alerts (default `false`) |

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness check |
| `GET` | `/analytics/anomalies` | List anomalies in rolling window |
| `GET` | `/analytics/heatmap` | Behaviour heatmap per user/action |
| `GET` | `/analytics/risk` | Risk scores |
| `GET` | `/analytics/reviews` | Privileged action review queue |
| `POST` | `/analytics/reviews/{id}/approve` | Approve reviewed action |

## Anomaly Detection Algorithm

Uses z-score over a 24-hour rolling window per user:

```
z = (observed_count - μ) / σ
```

- `μ` = mean hourly event count for that user over the past 7 days
- `σ` = standard deviation
- **z ≥ 3.0** triggers an anomaly record (configurable via `NSELF_AUDIT_ANOMALY_ZSCORE`)

Special rules (no z-score required):
- **bulk_delete**: ≥50 DELETE events in 60 seconds
- **after_hours_admin**: admin write between 20:00–06:00 UTC

## Heatmap Data

Returns a 7×24 matrix (day of week × hour of day) per user, showing normalised event frequency (0.0–1.0). Useful for identifying after-hours or weekend spikes.

## Risk Score

Composite 0–100 score per session:

| Factor | Weight |
|---|---|
| Z-score deviation | 40% |
| Privilege level (admin/normal) | 35% |
| Time-of-day (after-hours) | 25% |

## Multi-App Isolation

All tables include `source_account_id TEXT NOT NULL DEFAULT 'primary'` per the nSelf Multi-Tenant Convention Wall.

## Database Tables

- `np_audit_log` — source events (written by `audit` plugin)
- `np_audit_anomalies` — detected anomalies
- `np_audit_heatmap_cache` — aggregated heatmap data
- `np_audit_privileged_reviews` — privileged action queue

## Notes

This plugin requires the `audit` plugin to be installed and writing to `np_audit_log`. The analytics plugin is read-heavy; the `audit` plugin is write-heavy. They can run independently.
