# Plugin: nself-synthetic-monitor

Browser and API synthetic checks that simulate user flows and validate endpoint behavior on a schedule.

**Bundle:** ɳSentry ($0.99/mo or $9.99/yr) · Port: 3836

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-synthetic-monitor
nself build
nself start
```

---

## Schema

| Table | Purpose |
|---|---|
| `np_synthetic_flows` | Synthetic check definitions (URL, steps, assertions) |
| `np_synthetic_runs` | Execution results per flow run |

---

## HTTP API

All endpoints proxy through `api.nself.org` via Nginx. Internal binding: `127.0.0.1:3836`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | none | Health check |
| GET | `/ready` | none | Readiness probe |
| GET | `/api/v1/flows` | bearer | List all synthetic flows |
| POST | `/api/v1/flows` | bearer | Create a synthetic flow |
| GET | `/api/v1/flows/{id}` | bearer | Get flow definition |
| PATCH | `/api/v1/flows/{id}` | bearer | Update flow |
| DELETE | `/api/v1/flows/{id}` | bearer | Remove flow |
| POST | `/api/v1/flows/{id}/run` | bearer | Trigger an on-demand run |
| GET | `/api/v1/runs` | bearer | Query run history |

Auth uses R3-PATTERN: Hasura JWT + Nginx `auth_request` + plugin bearer token.

```bash
# Trigger on-demand synthetic run
curl -X POST -H "Authorization: Bearer $PLUGIN_SECRET" \
     https://api.nself.org/nself-synthetic-monitor/api/v1/flows/my-flow-id/run
```

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `3836` | Plugin listen port |

---

## Security

- Binds to `127.0.0.1` only. Never exposed directly.
- Requires active ɳSentry bundle license (`nself_pro_...`).
- All endpoints require bearer token auth.
- Row-level isolation via `source_account_id` column.

---

## See also

- [[plugin-nself-slo-tracker]] — use synthetic results in SLO calculations
- [[plugin-nself-uptime-monitor]] — complementary real-time uptime probes
- [[plugin-nself-alert-router]] — route synthetic failure alerts
- [[Plugin-Overview]] · [[Home]]
