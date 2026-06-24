# Audit Analytics Plugin

> Advanced audit log analytics: anomaly detection, user behaviour heatmaps, privileged-action review queue, and webhook/email alerts. **Pro plugin — ɳSelf+ required.**

## Tier Required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|----------------------|
| Free | $0 | $0 | No |
| Any bundle | $0.99/mo | $9.99/yr | No (unbundled) |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** ɳSelf+. This plugin is unbundled (not in any named bundle). Purchase ɳSelf+ for access.

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install audit-analytics
nself build
```

## What It Does

Runs continuous analysis over `np_audit_log` to surface security-relevant patterns that raw log queries miss.

**Anomaly detection** uses a z-score algorithm against a rolling 7-day baseline per `(user, action, hour)` tuple. Five anomaly types are detected:

- `freq_spike` — action frequency > z-score threshold vs 7-day baseline (default: 3.0σ)
- `bulk_delete` — ≥50 delete events in 60 seconds
- `new_ip` — user logging in from an IP not seen in past 7 days
- `after_hours_admin` — privileged action in configured after-hours UTC window (default: 20:00–06:00)
- `impossible_travel` — geolocation inconsistency between consecutive sessions

Severity buckets: z-score 3.0–4.9 → `medium`, 5.0–7.9 → `high`, ≥8.0 → `critical`.

**Heatmaps** aggregate action counts by `(user, hour_of_week)` into `np_audit_heatmap` (materialized view). Refreshed on a configurable interval (default: 1 hour via background worker).

**Privileged-action review queue** routes `role_grant`, `plugin_install`, `schema_alter`, and `gdpr_delete` to `np_audit_privileged_reviews`. Items become overdue after a configurable TTL (default: 24 h). The `privileged_review.overdue` webhook fires on expiry.

**Alerts** deliver anomaly notifications via webhook (`NSELF_AUDIT_ALERT_WEBHOOK`) or email (`NSELF_AUDIT_ALERT_EMAIL`). Real-time mode (`NSELF_AUDIT_ANALYTICS_REALTIME=true`) enables sub-second delivery (Enterprise).

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | Auto-provided by nSelf |
| `NSELF_AUDIT_ANALYTICS` | Yes | — | Set to `true` to enable (license gate) |
| `AUDIT_ANALYTICS_PORT` | No | `3714` | HTTP listener port (**conflict — see below**) |
| `AUDIT_ANALYTICS_SHARED_SECRET` | No | — | Bearer token (open-dev mode if unset) |
| `AUDIT_ANALYTICS_LOG_LEVEL` | No | `info` | `debug` / `info` / `warn` / `error` |
| `NSELF_AUDIT_ANOMALY_ZSCORE` | No | `3.0` | Minimum z-score threshold |
| `NSELF_AUDIT_HEATMAP_REFRESH` | No | `3600` | Heatmap refresh interval (seconds) |
| `NSELF_AUDIT_PRIVILEGED_REVIEW_TTL` | No | `86400` | Seconds until review overdue (24 h) |
| `NSELF_AUDIT_ALERT_WEBHOOK` | No | — | Webhook URL for anomaly alerts |
| `NSELF_AUDIT_ALERT_EMAIL` | No | — | Email for anomaly notifications |
| `NSELF_AUDIT_ANALYTICS_REALTIME` | No | `false` | Real-time alerts (Enterprise) |

## Ports

| Port | Purpose |
|------|---------|
| 3714 | Audit Analytics REST API (default — **conflict with voice plugin**) |

## ⚠ Port Conflict — Port 3714 (F10 Registry Issue)

**Both `audit-analytics` and `voice` default to port 3714.** Running both in the same nSelf deployment without overriding one will cause a startup failure.

| Plugin | Env Var | Default Port |
|--------|---------|-------------|
| audit-analytics | `AUDIT_ANALYTICS_PORT` | **3714** |
| voice | `VOICE_PORT` | **3714** |

**Workaround:** override `AUDIT_ANALYTICS_PORT` before running `nself build`:

```bash
nself env set AUDIT_ANALYTICS_PORT=3715   # use any port not in F10-PORT-REGISTRY
```

**Status:** This conflict is tracked in F10-PORT-REGISTRY and requires a registry-level deduplication. Do not run both plugins on port 3714. A dedicated port will be assigned to one of these plugins in a future registry update.

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_audit_anomalies` | Scored anomaly records |
| `np_audit_privileged_reviews` | Privileged-action review queue |
| `np_audit_heatmap` | Materialized view (action × hour_of_week) |

Both `np_audit_anomalies` and `np_audit_privileged_reviews` are tenant-isolated via `tenant_id UUID` with Hasura row filter `{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}`.

## API

```
GET  /health                              — Health check
GET  /audit/analytics/anomalies           — List anomalies
GET  /audit/analytics/anomalies/{id}      — Get anomaly
PATCH /audit/analytics/anomalies/{id}     — Set disposition
GET  /audit/analytics/heatmap             — Heatmap data
GET  /audit/analytics/top-actors          — Top actors by count
GET  /audit/privileged-reviews            — List review items
POST /audit/privileged-reviews/{id}       — Submit review
GET  /audit/privileged-reviews/overdue    — Overdue reviews
POST /audit/analytics/refresh             — Force heatmap refresh
GET  /audit/analytics/status              — Status + counts
```

## Webhooks

| Event | Trigger |
|-------|---------|
| `anomaly.detected` | Anomaly exceeds threshold |
| `anomaly.reviewed` | Reviewer sets disposition |
| `privileged_review.overdue` | Review not completed within TTL |

## Docker Hub

```bash
docker pull nself-org/plugin-audit-analytics:latest
```

Image: `nself-org/plugin-audit-analytics:latest`

## Notes

- Compliance: designed for SOC 2, PCI-DSS, HIPAA, ISO 27001 audit trail requirements.
- Anomaly scoring runs entirely within your Postgres instance — no audit data leaves the deployment except via configured alert targets.
- Scoring runs every 5 minutes when paired with the `cron` plugin. Without `cron`, call `POST /audit/analytics/refresh` on your own schedule.
- The heatmap background worker refreshes automatically at the configured interval regardless of cron.

## Source

Source-available (license required to run): [`plugins-pro/paid/audit-analytics/`](https://github.com/nself-org/plugins-pro/tree/main/paid/audit-analytics)
