# Plugin: nself-uptime-monitor

HTTP uptime monitoring with configurable probe intervals, target management, and Alertmanager integration.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3831

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-uptime-monitor
nself build
nself start
```

---

## Schema

| Table | Purpose |
|---|---|
| `np_uptime_targets` | Monitored URLs and configuration |
| `np_uptime_probes` | Probe definitions and schedules |
| `np_uptime_results` | Historical probe results |

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3831`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| POST | `/probe` | bearer | Trigger a manual probe |
| GET | `/api/v1/targets` | bearer | List all targets |
| POST | `/api/v1/targets` | bearer | Create a target |
| GET | `/api/v1/targets/{id}` | bearer | Get target details |
| DELETE | `/api/v1/targets/{id}` | bearer | Remove a target |
| GET | `/api/v1/results` | bearer | Query probe results |
| GET | `/api/v1/stats` | bearer | Uptime statistics |

Auth uses R3-PATTERN: Hasura JWT + Nginx `auth_request` + plugin bearer token.

```bash
curl -H "Authorization: Bearer $PLUGIN_SECRET" \
     https://api.nself.org/nself-uptime-monitor/api/v1/targets
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3831` | Plugin listen port |
| `UPTIME_PROBE_INTERVAL_SECONDS` | no | `60` | How often probes run |
| `UPTIME_PROBE_TIMEOUT_SECONDS` | no | `10` | Per-probe timeout |
| `UPTIME_ALERTMANAGER_URL` | no | — | Alertmanager endpoint for firing alerts |
| `UPTIME_PROMETHEUS_URL` | no | — | Prometheus endpoint for metrics |
| `UPTIME_MAX_CONCURRENT_PROBES` | no | — | Parallel probe limit |

Required services: Prometheus, Alertmanager.

---

## Security

- Binds to `127.0.0.1` only. Never exposed directly.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- All write endpoints require bearer token auth.
- Row-level isolation via `source_account_id` column.

---

## See also

- [[plugin-nself-status-page]] — public status page surface
- [[plugin-nself-alert-router]] — alert routing and escalation
- [[plugin-nself-slo-tracker]] — SLO definitions and burn rates
- [[Plugin-Overview]] · [[Home]]
