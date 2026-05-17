# Plugin: nself-slo-tracker

Service Level Objective tracking with burn rate calculation, SLO compliance reporting, and error budget management.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3835

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-slo-tracker
nself build
nself start
```

---

## Schema

| Table | Purpose |
|---|---|
| `np_slos` | SLO definitions with targets and windows |
| `np_slo_snapshots` | Point-in-time SLO compliance snapshots |
| `np_slo_reports` | Aggregated SLO reports per period |

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3835`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| GET | `/api/v1/slos` | bearer | List all SLOs |
| POST | `/api/v1/slos` | bearer | Create an SLO |
| GET | `/api/v1/slos/{id}` | bearer | Get SLO detail and current status |
| PATCH | `/api/v1/slos/{id}` | bearer | Update SLO targets |
| DELETE | `/api/v1/slos/{id}` | bearer | Remove SLO |
| GET | `/api/v1/slos/{id}/burn-rate` | bearer | Current burn rate |
| GET | `/api/v1/slos/{id}/snapshots` | bearer | Historical snapshots |
| GET | `/api/v1/reports` | bearer | Aggregated SLO compliance reports |

Auth uses R3-PATTERN: Hasura JWT + Nginx `auth_request` + plugin bearer token.

```bash
curl -H "Authorization: Bearer $PLUGIN_SECRET" \
     https://api.nself.org/nself-slo-tracker/api/v1/slos
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3835` | Plugin listen port |

---

## Security

- Binds to `127.0.0.1` only. Never exposed directly.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- All endpoints require bearer token auth.
- Row-level isolation via `source_account_id` column.

---

## See also

- [[plugin-nself-uptime-monitor]] — uptime data source for SLO calculation
- [[plugin-nself-alert-router]] — alert when SLO burn rate is critical
- [[plugin-nself-synthetic-monitor]] — synthetic check data for SLOs
- [[Plugin-Overview]] · [[Home]]
