# Plugin: nself-errors

JavaScript and server-side error tracking with stack traces, release tracking, source map support, and breadcrumb context.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3838

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-errors
nself build
nself start
```

---

## Schema

| Table | Purpose |
|---|---|
| `np_errors_events` | Individual error occurrences with stack traces and context |
| `np_errors_groups` | Deduplicated error groups (fingerprint-based) |
| `np_errors_releases` | Release records for error-to-release correlation |
| `np_errors_sourcemaps` | Uploaded source map files for stack trace symbolication |
| `np_errors_breadcrumbs` | Breadcrumb trail leading up to each error event |

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3838`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| POST | `/api/v1/ingest/error` | none | Ingest an error event from client SDK |
| GET | `/api/v1/errors` | bearer | List error groups |
| GET | `/api/v1/errors/{id}` | bearer | Get error group detail with events |
| GET | `/api/v1/releases` | bearer | List releases |
| POST | `/api/v1/releases` | bearer | Create a release record |
| POST | `/api/v1/sourcemaps` | bearer | Upload source map for a release |

Auth uses R3-PATTERN on management endpoints. The ingest endpoint accepts anonymous data from frontend and backend SDKs.

```bash
# List error groups
curl -H "Authorization: Bearer $PLUGIN_SECRET" \
     "https://api.nself.org/nself-errors/api/v1/errors?limit=20"
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3838` | Plugin listen port |

---

## Security

- Ingest endpoint is intentionally public — SDKs must reach it from browsers and servers without auth.
- Management endpoints (list, releases, source maps) require bearer token auth.
- Binds to `127.0.0.1`. Nginx routes the ingest endpoint at the edge with rate limiting.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- Source maps may contain sensitive path information — apply access controls in Hasura.

---

## See also

- [[plugin-nself-rum]] — Real User Monitoring companion
- [[plugin-nself-crash]] — native crash reporting (mobile/desktop)
- [[plugin-nself-alert-router]] — alert on error rate spikes
- [[Plugin-Overview]] · [[Home]]
