# Plugin: nself-crash

Native crash reporting for mobile and desktop apps with symbolication, crash grouping, and session context.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3841

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-crash
nself build
nself start
```

---

## Schema

| Table | Purpose |
|---|---|
| `np_crash_reports` | Individual crash reports with stack traces, device info, and OS details |
| `np_crash_groups` | Deduplicated crash groups (fingerprint-based) |
| `np_crash_sessions` | App session records for crash-free session rate calculation |

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3841`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| POST | `/api/v1/ingest/crash` | none | Ingest a crash report from SDK |
| POST | `/api/v1/ingest/session` | none | Ingest a session heartbeat |
| GET | `/api/v1/crashes` | bearer | List crash groups |
| GET | `/api/v1/crashes/{id}` | bearer | Get crash group with reports |
| GET | `/api/v1/crashes/{id}/reports` | bearer | List individual reports in a group |
| GET | `/api/v1/sessions` | bearer | Query session data |
| GET | `/api/v1/stats` | bearer | Crash-free session rate and summary stats |
| POST | `/api/v1/dsyms` | bearer | Upload dSYM / symbol file for symbolication |

Auth uses R3-PATTERN on management endpoints. Ingest endpoints accept anonymous data from native SDKs.

```bash
# Get crash-free session rate
curl -H "Authorization: Bearer $PLUGIN_SECRET" \
     "https://api.nself.org/nself-crash/api/v1/stats?period=7d"
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3841` | Plugin listen port |

---

## Security

- Ingest endpoints are intentionally public — native SDKs must reach them without auth.
- Management endpoints (crash groups, sessions, stats, symbol upload) require bearer token auth.
- Binds to `127.0.0.1`. Nginx routes ingest endpoints at the edge with rate limiting.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- dSYM and symbol files may contain sensitive path information — restrict access in Hasura.
- Row-level isolation via `source_account_id` column.

---

## See also

- [[plugin-nself-errors]] — JavaScript and server-side error tracking companion
- [[plugin-nself-rum]] — Real User Monitoring (sessions + vitals)
- [[plugin-nself-alert-router]] — alert on crash rate spikes
- [[Plugin-Overview]] · [[Home]]
