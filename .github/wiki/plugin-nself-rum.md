# Plugin: nself-rum

Real User Monitoring (RUM) with session tracking, Core Web Vitals, session replay, and distributed tracing breadcrumbs.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3837

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-rum
nself build
nself start
```

---

## Schema

| Table | Purpose |
|---|---|
| `np_rum_sessions` | User session records with metadata |
| `np_rum_vitals` | Core Web Vitals measurements (LCP, FID, CLS, TTFB) |
| `np_rum_replay_chunks` | Session replay event chunks |
| `np_rum_breadcrumbs` | User action and navigation breadcrumbs |
| `np_rum_dsns` | Data Source Names for frontend SDK initialization |

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3837`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| POST | `/api/v1/ingest/session` | none | Ingest session start event |
| POST | `/api/v1/ingest/vitals` | none | Ingest Web Vitals measurements |
| POST | `/api/v1/ingest/replay` | none | Ingest session replay chunk |
| POST | `/api/v1/ingest/breadcrumbs` | none | Ingest breadcrumb events |
| GET | `/api/v1/sessions` | bearer | Query session records |
| GET | `/api/v1/sessions/{id}` | bearer | Get session detail with replay |
| GET | `/api/v1/vitals` | bearer | Query Web Vitals data |
| GET | `/api/v1/dsns` | bearer | List project DSNs |
| POST | `/api/v1/dsns` | bearer | Create a project DSN |
| GET | `/api/v1/stats` | bearer | Aggregated performance stats |
| GET | `/api/v1/replay/{session_id}` | bearer | Fetch full session replay |

Auth uses R3-PATTERN on management endpoints. Ingest endpoints accept anonymous data from frontend SDK.

```bash
# Query recent sessions
curl -H "Authorization: Bearer $PLUGIN_SECRET" \
     "https://api.nself.org/nself-rum/api/v1/sessions?limit=20"
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3837` | Plugin listen port |

---

## Frontend SDK

Initialize in your frontend app using the DSN from the admin API:

```js
import { RumClient } from '@nself/rum-js'
RumClient.init({ dsn: 'https://api.nself.org/nself-rum/ingest?token=...' })
```

---

## Security

- Ingest endpoints are intentionally public (browsers must reach them without auth).
- Management endpoints require bearer token auth.
- Binds to `127.0.0.1`. Nginx routes ingest endpoints at the edge with rate limiting.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- Session replay data is sensitive — apply row-level access controls in Hasura.

---

## See also

- [[plugin-nself-errors]] — JavaScript error tracking companion
- [[plugin-nself-slo-tracker]] — feed RUM vitals into SLO calculations
- [[plugin-nself-alert-router]] — alert on degraded Core Web Vitals
- [[Plugin-Overview]] · [[Home]]
