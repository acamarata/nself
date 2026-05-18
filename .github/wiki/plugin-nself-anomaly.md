# Plugin: nself-anomaly

Statistical anomaly detection on metrics and time-series data with configurable detectors and alert thresholds.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3842

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-anomaly
nself build
nself start
```

---

## Schema

| Table | Purpose |
|---|---|
| `np_anomaly_detectors` | Detector definitions with metric source, algorithm, and threshold config |
| `np_anomaly_data_points` | Ingested data points and scored anomaly results |

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3842`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| POST | `/api/v1/ingest` | bearer | Ingest data points for scoring |
| GET | `/api/v1/detectors` | bearer | List detectors |
| POST | `/api/v1/detectors` | bearer | Create a detector |
| GET | `/api/v1/detectors/{id}` | bearer | Get detector config and status |
| PATCH | `/api/v1/detectors/{id}` | bearer | Update detector thresholds |
| DELETE | `/api/v1/detectors/{id}` | bearer | Remove detector |
| GET | `/api/v1/anomalies` | bearer | Query detected anomalies |
| GET | `/api/v1/anomalies/{id}` | bearer | Get anomaly detail with scoring context |
| GET | `/api/v1/stats` | bearer | Detection summary and false-positive rate |

Auth uses R3-PATTERN: Hasura JWT + Nginx `auth_request` + plugin bearer token.

```bash
# Ingest a data point for scoring
curl -X POST -H "Authorization: Bearer $PLUGIN_SECRET" \
     -H "Content-Type: application/json" \
     -d '{"detector_id":"my-detector","value":142.7,"timestamp":"2026-05-17T10:00:00Z"}' \
     https://api.nself.org/nself-anomaly/api/v1/ingest
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3842` | Plugin listen port |

---

## Security

- Binds to `127.0.0.1` only. Never exposed directly.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- All endpoints require bearer token auth.
- Row-level isolation via `source_account_id` column.

---

## See also

- [[plugin-nself-slo-tracker]] — feed anomaly events into SLO burn rate
- [[plugin-nself-alert-router]] — route anomaly alerts to channels
- [[plugin-nself-rum]] — RUM vitals as an anomaly detection input source
- [[Plugin-Overview]] · [[Home]]
