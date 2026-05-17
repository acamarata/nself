# Plugin: nself-alert-router

Alert routing and escalation engine that dispatches notifications across channels based on configurable rules.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3834

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-alert-router
nself build
nself start
```

---

## Schema

Alert routing rules and notification channel configs are stored as JSON within the plugin's own database. No `np_*` Hasura tables.

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3834`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| POST | `/api/v1/alerts` | bearer | Receive and route an alert |
| GET | `/api/v1/routes` | bearer | List routing rules |
| POST | `/api/v1/routes` | bearer | Create routing rule |
| DELETE | `/api/v1/routes/{id}` | bearer | Remove routing rule |
| GET | `/api/v1/channels` | bearer | List notification channels |
| POST | `/api/v1/channels` | bearer | Add notification channel |
| POST | `/api/v1/silence` | bearer | Mute alerts matching a pattern |

Auth uses R3-PATTERN: Hasura JWT + Nginx `auth_request` + plugin bearer token.

```bash
# Send an alert
curl -X POST -H "Authorization: Bearer $PLUGIN_SECRET" \
     -H "Content-Type: application/json" \
     -d '{"name":"high_error_rate","labels":{"service":"api"},"severity":"critical"}' \
     https://api.nself.org/nself-alert-router/api/v1/alerts
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3834` | Plugin listen port |

Supports integrating with Alertmanager, PagerDuty, Slack, or webhook endpoints via channel config.

---

## Security

- Binds to `127.0.0.1` only. Never exposed directly.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- All endpoints require bearer token auth.

---

## See also

- [[plugin-nself-uptime-monitor]] — source of uptime alerts
- [[plugin-nself-incident-mgmt]] — incident lifecycle management
- [[plugin-nself-oncall]] — on-call escalation
- [[Plugin-Overview]] · [[Home]]
