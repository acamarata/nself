# Plugin: nself-oncall

On-call rotation scheduling with escalation policies, overrides, and notification delivery.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3840

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-oncall
nself build
nself start
```

---

## Schema

On-call schedules and escalation policies are stored in the plugin's internal database. No `np_*` Hasura tables.

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3840`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| GET | `/api/v1/schedules` | bearer | List on-call schedules |
| POST | `/api/v1/schedules` | bearer | Create a schedule |
| GET | `/api/v1/schedules/{id}` | bearer | Get schedule with current on-call person |
| PATCH | `/api/v1/schedules/{id}` | bearer | Update schedule |
| DELETE | `/api/v1/schedules/{id}` | bearer | Remove schedule |
| GET | `/api/v1/schedules/{id}/oncall` | bearer | Get current on-call person for a schedule |
| POST | `/api/v1/overrides` | bearer | Create a temporary override |
| DELETE | `/api/v1/overrides/{id}` | bearer | Remove an override |
| GET | `/api/v1/policies` | bearer | List escalation policies |
| POST | `/api/v1/policies` | bearer | Create an escalation policy |

Auth uses R3-PATTERN: Hasura JWT + Nginx `auth_request` + plugin bearer token.

```bash
# Get the current on-call person
curl -H "Authorization: Bearer $PLUGIN_SECRET" \
     https://api.nself.org/nself-oncall/api/v1/schedules/my-schedule/oncall
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3840` | Plugin listen port |

---

## Security

- Binds to `127.0.0.1` only. Never exposed directly.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- All endpoints require bearer token auth.

---

## See also

- [[plugin-nself-alert-router]] — routes alerts to the current on-call person
- [[plugin-nself-incident-mgmt]] — links incidents to on-call schedules
- [[plugin-nself-slo-tracker]] — burn rate alerts trigger on-call escalation
- [[Plugin-Overview]] · [[Home]]
