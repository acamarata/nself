# Plugin: nself-cron-monitor

Scheduled job monitoring that detects missed, late, and failing cron runs with configurable alert thresholds.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3839

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-cron-monitor
nself build
nself start
```

---

## Schema

Cron monitor tracks job execution state in its own internal database. No `np_*` Hasura tables.

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3839`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| POST | `/api/v1/ping/{monitor_id}` | none | Record a successful job heartbeat |
| POST | `/api/v1/ping/{monitor_id}/start` | none | Mark a job run as started |
| POST | `/api/v1/ping/{monitor_id}/fail` | none | Mark a job run as failed |
| GET | `/api/v1/monitors` | bearer | List all cron monitors |
| POST | `/api/v1/monitors` | bearer | Create a cron monitor |
| GET | `/api/v1/monitors/{id}` | bearer | Get monitor status and history |
| PATCH | `/api/v1/monitors/{id}` | bearer | Update monitor schedule or thresholds |
| DELETE | `/api/v1/monitors/{id}` | bearer | Remove a monitor |

Auth uses R3-PATTERN on management endpoints. Ping endpoints are public — they receive heartbeats from your cron jobs.

```bash
# Record a successful heartbeat from your cron job
curl -X POST \
     "https://api.nself.org/nself-cron-monitor/api/v1/ping/my-monitor-id"
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3839` | Plugin listen port |

---

## Usage

After creating a monitor via the API, add a ping call to your cron script:

```bash
#!/bin/bash
# At the end of your cron job, signal success
curl -s -X POST "https://api.nself.org/nself-cron-monitor/api/v1/ping/$MONITOR_ID"
```

If the expected ping does not arrive within the configured grace period, the plugin triggers an alert via [[plugin-nself-alert-router]].

---

## Security

- Ping endpoints are intentionally public — cron jobs on any host must reach them without auth.
- Management endpoints require bearer token auth.
- Binds to `127.0.0.1`. Nginx routes ping endpoints at the edge with rate limiting.
- Requires active ɳSentry bundle license (`nself_pro_...`).

---

## See also

- [[plugin-nself-alert-router]] — route missed-job alerts
- [[plugin-nself-slo-tracker]] — roll cron reliability into SLO calculations
- [[plugin-nself-incident-mgmt]] — escalate chronic cron failures to incidents
- [[Plugin-Overview]] · [[Home]]
