# Plugin: audit-log

Append-only audit log for security-relevant events. Captures auth activity, privilege changes, secret access, and plugin install/uninstall. Queryable from Admin with filters by event type, actor, severity, and time range.

**Security-Always-Free:** This plugin has no license requirement. Audit logging is a core security feature available to all nSelf users at no cost.

## Install

```bash
nself plugin install audit-log
nself build
nself start
```

## Schema: np_auditlog_events

| Column | Type | Description |
|---|---|---|
| `id` | TEXT (PK) | Unique event ID (UUID) |
| `source_account_id` | TEXT DEFAULT 'primary' | Multi-app isolation — which app in this nSelf deploy generated the event |
| `actor_user_id` | TEXT | User who performed the action (null for system events) |
| `actor_type` | TEXT | `user`, `system`, `plugin`, `api_key` |
| `event_type` | TEXT | Dot-notation event identifier, e.g. `auth.login.success`, `plugin.install` |
| `resource_type` | TEXT | What was acted on: `user`, `secret`, `plugin`, `table`, etc. |
| `resource_id` | TEXT | ID of the affected resource |
| `ip_address` | TEXT | Client IP at time of event |
| `user_agent` | TEXT | HTTP User-Agent header |
| `metadata` | JSONB | Structured event-specific data |
| `severity` | TEXT | `info`, `warn`, `error`, `critical` |
| `created_at` | TIMESTAMPTZ | Event timestamp (partitioned by this column) |

The table is partitioned by `RANGE (created_at)` for scalable long-term retention.

### Append-only enforcement

RLS policies block UPDATE and DELETE on `np_auditlog_events`. Events are immutable once written. This is enforced at the database level — not just the application layer.

## HTTP API

All endpoints require the `X-Plugin-Secret` header (set to `PLUGIN_INTERNAL_SECRET`).

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check — returns `{"status":"ok"}` |
| `POST` | `/events` | Write one or more audit events |
| `GET` | `/events` | Query events with filters |
| `GET` | `/events/{id}` | Fetch a single event by ID |
| `GET` | `/events/export` | Export events as JSON or CSV |
| `GET` | `/admin/events` | Admin query — also accepts `HASURA_GRAPHQL_ADMIN_SECRET` header |

### POST /events — write an event

```bash
curl -X POST http://127.0.0.1:3308/events \
  -H "X-Plugin-Secret: $PLUGIN_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "auth.login.success",
    "actor_user_id": "user-abc",
    "actor_type": "user",
    "resource_type": "session",
    "resource_id": "session-xyz",
    "severity": "info",
    "ip_address": "192.168.1.100",
    "metadata": {"mfa_used": true}
  }'
```

### GET /events — query with filters

| Query param | Description |
|---|---|
| `event_type` | Filter by event type (exact or prefix) |
| `actor_user_id` | Filter by actor |
| `actor_type` | Filter by actor type |
| `severity` | Filter by severity level |
| `resource_type` | Filter by resource type |
| `resource_id` | Filter by resource ID |
| `from` | Start of time range (ISO 8601) |
| `to` | End of time range (ISO 8601) |
| `limit` | Max results (default 100, max 1000) |
| `cursor` | Pagination cursor from previous response |
| `source_account_id` | Filter by app (multi-app deployments) |

```bash
# Get all auth events from the last 24 hours
curl "http://127.0.0.1:3308/events?event_type=auth&from=2026-05-16T00:00:00Z" \
  -H "X-Plugin-Secret: $PLUGIN_INTERNAL_SECRET"
```

### GET /events/export — bulk export

```bash
# Export as CSV
curl "http://127.0.0.1:3308/events/export?format=csv&from=2026-05-01T00:00:00Z" \
  -H "X-Plugin-Secret: $PLUGIN_INTERNAL_SECRET" \
  -o audit-may-2026.csv

# Export as JSON
curl "http://127.0.0.1:3308/events/export?format=json&severity=critical" \
  -H "X-Plugin-Secret: $PLUGIN_INTERNAL_SECRET"
```

## Configuration

| Env var | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `PORT` | No | Plugin port (default: 3308) |
| `HASURA_GRAPHQL_ADMIN_SECRET` | No | Enables the `/admin/events` endpoint for Hasura-level admin queries |
| `PLUGIN_INTERNAL_SECRET` | No | Shared secret for `X-Plugin-Secret` header authentication |

## Common event types

| Event type | Triggered by |
|---|---|
| `auth.login.success` | Successful login |
| `auth.login.failure` | Failed login attempt |
| `auth.logout` | User logout |
| `auth.mfa.enabled` | MFA turned on |
| `auth.mfa.disabled` | MFA turned off |
| `auth.password.changed` | Password change |
| `plugin.install` | Plugin installed |
| `plugin.uninstall` | Plugin removed |
| `secret.accessed` | Vault secret read |
| `secret.rotated` | Vault secret rotated |
| `privilege.granted` | Role/permission added |
| `privilege.revoked` | Role/permission removed |

## Querying from Admin

The nSelf Admin UI (localhost:3021) includes an Audit Log viewer under Security. You can filter by event type, actor, severity, and date range without writing SQL.

For custom queries, connect directly to Postgres:

```sql
SELECT event_type, actor_user_id, severity, created_at, metadata
FROM np_auditlog_events
WHERE event_type LIKE 'auth%'
  AND created_at > NOW() - INTERVAL '7 days'
ORDER BY created_at DESC
LIMIT 100;
```

## Security notes

- Port 3308 binds to `127.0.0.1` — never exposed externally.
- Events are immutable at the database level (RLS blocks UPDATE/DELETE).
- No license required — audit logging is free for all nSelf users (Security-Always-Free).
- `nself doctor --deep` includes check `PERM-RLS-01` to verify append-only RLS is active.

## See also

- [Security overview](Security.md)
- [plugin-nself-scan](plugin-nself-scan.md) — vulnerability scanner (also free)
