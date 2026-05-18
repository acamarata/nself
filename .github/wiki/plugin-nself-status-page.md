# Plugin: nself-status-page

Public status page for your nSelf instance, showing uptime history, active incidents, and service health.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3832

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-status-page
nself build
nself start
```

---

## Schema

This plugin uses incident and probe data from companion ɳSentry plugins. No dedicated tables.

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3832`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| GET | `/api/v1/status` | none | Public status summary |
| GET | `/api/v1/incidents` | none | Active and recent incidents |
| POST | `/api/v1/incidents` | bearer | Create a new incident |
| PATCH | `/api/v1/incidents/{id}` | bearer | Update incident status |
| GET | `/api/v1/services` | none | Service health list |

Auth on write endpoints uses R3-PATTERN: Hasura JWT + Nginx `auth_request` + plugin bearer token.

```bash
# Public status endpoint (no auth)
curl https://api.nself.org/nself-status-page/api/v1/status

# Create incident (requires auth)
curl -X POST -H "Authorization: Bearer $PLUGIN_SECRET" \
     -H "Content-Type: application/json" \
     -d '{"title":"Database slowdown","severity":"minor"}' \
     https://api.nself.org/nself-status-page/api/v1/incidents
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3832` | Plugin listen port |

---

## Security

- Binds to `127.0.0.1` only. Never exposed directly.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- Public read endpoints have no auth requirement by design (status pages are public).
- Write endpoints require bearer token auth.

---

## See also

- [[plugin-nself-uptime-monitor]] — uptime probe data source
- [[plugin-nself-incident-mgmt]] — structured incident management
- [[plugin-nself-alert-router]] — alert routing and escalation
- [[Plugin-Overview]] · [[Home]]
