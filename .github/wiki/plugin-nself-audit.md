# Plugin: nself-audit

Structured audit log for all user actions and system events across your nSelf stack.

**Bundle:** Free tier — no license required · Port: 3843

---

## Install

```bash
nself plugin install nself-audit
nself build
nself start
```

No license key needed. nself-audit is part of the free tier and ships as a standard plugin.

---

## Schema

| Table | Purpose |
|---|---|
| `np_audit_events` | Append-only log of user and system events with actor, resource, action, and metadata |

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3843`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| POST | `/api/v1/events` | bearer | Record an audit event |
| GET | `/api/v1/events` | bearer | Query audit events with filters |
| GET | `/api/v1/events/{id}` | bearer | Get a single event by ID |
| GET | `/api/v1/actors/{actor_id}/events` | bearer | All events for a specific actor |
| GET | `/api/v1/stats` | bearer | Event volume and summary statistics |

Auth uses R3-PATTERN: Hasura JWT + Nginx `auth_request` + plugin bearer token.

```bash
# Query recent audit events
curl -H "Authorization: Bearer $PLUGIN_SECRET" \
     "https://api.nself.org/nself-audit/api/v1/events?limit=50&sort=desc"
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3843` | Plugin listen port |

---

## Security

- Binds to `127.0.0.1` only. Never exposed directly.
- No license required — available on the free tier.
- All endpoints require bearer token auth.
- `np_audit_events` is append-only by design — delete permissions are restricted.
- Row-level isolation via `source_account_id` column.

---

## See also

- [[plugin-nself-incident-mgmt]] — attach audit trail to incidents
- [[plugin-nself-alert-router]] — alert on suspicious audit patterns
- [[plugin-nself-slo-tracker]] — include audit volume in SLO calculations
- [[Plugin-Overview]] · [[Home]]
