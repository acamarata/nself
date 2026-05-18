# Plugin: nself-incident-mgmt

Structured incident lifecycle management with timeline tracking and runbook execution.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3833

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-incident-mgmt
nself build
nself start
```

---

## Schema

| Table | Purpose |
|---|---|
| `np_incidents` | Incident records with severity, status, and ownership |
| `np_incident_timeline` | Ordered event log per incident |
| `np_runbooks` | Stored runbook definitions for common incident types |

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3833`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| GET | `/api/v1/incidents` | bearer | List all incidents |
| POST | `/api/v1/incidents` | bearer | Open a new incident |
| GET | `/api/v1/incidents/{id}` | bearer | Get incident detail |
| PATCH | `/api/v1/incidents/{id}` | bearer | Update status or ownership |
| POST | `/api/v1/incidents/{id}/timeline` | bearer | Add timeline event |
| GET | `/api/v1/runbooks` | bearer | List runbooks |
| POST | `/api/v1/runbooks` | bearer | Create runbook |

Auth uses R3-PATTERN: Hasura JWT + Nginx `auth_request` + plugin bearer token.

```bash
curl -H "Authorization: Bearer $PLUGIN_SECRET" \
     https://api.nself.org/nself-incident-mgmt/api/v1/incidents
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3833` | Plugin listen port |

---

## Security

- Binds to `127.0.0.1` only. Never exposed directly.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- All endpoints require bearer token auth.
- Row-level isolation via `source_account_id` column.

---

## See also

- [[plugin-nself-status-page]] — public status page surface
- [[plugin-nself-alert-router]] — alert routing and escalation
- [[plugin-nself-oncall]] — on-call rotation integration
- [[Plugin-Overview]] · [[Home]]
