# plugin-clawde

**ClawDE daemon integration backend** — manages workspace sessions, daemon health probes, and real-time event streaming for the ɳClawDE AI development environment.

| | |
|---|---|
| **Port** | 3847 |
| **Bundle** | ClawDE |
| **Tier** | Pro |
| **License** | Source-Available |
| **Version** | 1.0.0 |
| **Language** | Go |
| **Tables** | `np_clawde_daemon_status`, `np_clawde_sessions`, `np_clawde_events` |
| **Requires license** | Yes |

---

## What It Does

plugin-clawde is the backend component of the ClawDE bundle. It runs as a sidecar to the nSelf backend and:

1. **Proxies daemon health checks** — The ClawDE Flutter app talks to this plugin, which probes the local ClawDE daemon (an AI coding assistant process). Auth is required for all daemon probes so daemon reachability is never disclosed without a valid tenant.

2. **Manages workspace sessions** — Sessions represent an active ClawDE development context. Each session is scoped to a tenant (`tenant_id`) and has a configurable TTL (default 24 hours).

3. **Streams events via SSE** — Provides a `GET /clawde/sessions/{id}/events` endpoint that pushes session events as Server-Sent Events. Uses a `(created_at, id)` keyset cursor for correctness — UUID v4 PKs are random and cannot be compared lexicographically for pagination.

---

## Installation

```bash
nself plugin install plugin-clawde
```

Requires a ClawDE bundle license or ɳSelf+ subscription.

---

## Environment Variables

### Required

| Variable | Description |
|---|---|
| `DATABASE_URL` | Postgres connection string |
| `CLAWDE_DAEMON_URL` | Base URL of the ClawDE daemon process |
| `CLAWDE_DAEMON_TOKEN` | Bearer token for daemon authentication |

### Optional

| Variable | Default | Description |
|---|---|---|
| `PORT` | `3847` | HTTP listen port |
| `LOG_LEVEL` | `info` | Log verbosity (debug/info/warn/error) |
| `PLUGIN_INTERNAL_SECRET` | — | Plugin-to-plugin shared secret |
| `CLAWDE_DAEMON_TIMEOUT_SECS` | `10` | Daemon probe timeout in seconds |
| `CLAWDE_SESSION_TTL_HOURS` | `24` | Session idle TTL in hours |
| `CLAWDE_MAX_SESSIONS_PER_TENANT` | `10` | Max concurrent sessions per tenant |

---

## API Reference

All endpoints except `/health` require:
- `Authorization: Bearer <token>`
- `X-Hasura-Tenant-Id: <uuid>`

Both are checked independently. Providing only one returns 401.

### Public

#### `GET /health`

Liveness check. Returns 200 if the plugin service is running. Does NOT probe the ClawDE daemon.

```json
{
  "status": "ok",
  "time": "2026-06-24T12:00:00Z"
}
```

### Daemon

#### `GET /clawde/daemon/health`

Proxies a health probe to the ClawDE daemon. Returns daemon status.

```json
{
  "healthy": true,
  "version": "1.0.0",
  "message": ""
}
```

Returns 503 if the daemon is unreachable.

#### `GET /clawde/daemon/status`

Extended daemon diagnostics. Same schema as `/clawde/daemon/health` with additional message fields.

### Sessions

#### `POST /clawde/sessions`

Create a new workspace session.

**Request:**
```json
{
  "name": "my-project",
  "workspace_path": "/home/user/projects/my-project"
}
```

**Response:** 201
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "...",
  "name": "my-project",
  "workspace_path": "/home/user/projects/my-project",
  "status": "pending",
  "created_at": "2026-06-24T12:00:00Z",
  "updated_at": "2026-06-24T12:00:00Z",
  "expires_at": "2026-06-25T12:00:00Z"
}
```

#### `GET /clawde/sessions`

List all sessions for the current tenant, ordered by `created_at DESC`.

**Response:** 200 — array of session objects.

#### `GET /clawde/sessions/{id}`

Get a specific session. Returns 404 if the session doesn't exist or belongs to another tenant.

#### `DELETE /clawde/sessions/{id}`

Delete a session. Cascade-deletes all associated events. Returns 204.

### Event Streaming

#### `GET /clawde/sessions/{id}/events`

Opens an SSE stream for the session. Returns `Content-Type: text/event-stream`.

**Cursor query parameters:**
- `last_ts` — RFC3339Nano timestamp of the last received event
- `last_id` — UUID of the last received event (tiebreak for same-microsecond events)

**Example SSE frame:**
```
data: {"id":"abc...","session_id":"...","tenant_id":"...","event_type":"file_changed","payload":{"path":"/foo.go"},"created_at":"2026-06-24T12:00:00.001Z"}

```

**Cursor correctness note:**
The cursor uses `(created_at, id)` pagination with a SQL tiebreak:
```sql
WHERE created_at > $cursorTS
   OR (created_at = $cursorTS AND id > $cursorID)
```
This ensures no events are lost when multiple events share the same timestamp, which UUID-only cursors would incorrectly skip.

---

## Database Tables

### `np_clawde_daemon_status`

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | |
| `tenant_id` | UUID | Hasura RLS filter |
| `source_account_id` | TEXT | Multi-app isolation |
| `daemon_url` | TEXT | |
| `healthy` | BOOLEAN | |
| `version` | TEXT | |
| `last_checked_at` | TIMESTAMPTZ | |
| `error_message` | TEXT | |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | |

Unique constraint: `(tenant_id, daemon_url)`.

### `np_clawde_sessions`

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | |
| `tenant_id` | UUID | Hasura RLS filter |
| `source_account_id` | TEXT | Multi-app isolation |
| `name` | TEXT | |
| `workspace_path` | TEXT | |
| `daemon_session_id` | TEXT | ClawDE daemon's own session ref |
| `status` | TEXT | pending / active / closed |
| `metadata` | JSONB | |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | |
| `expires_at` | TIMESTAMPTZ | |

### `np_clawde_events`

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | |
| `session_id` | UUID | FK → `np_clawde_sessions(id)` CASCADE |
| `tenant_id` | UUID | Hasura RLS filter |
| `source_account_id` | TEXT | Multi-app isolation |
| `event_type` | TEXT | |
| `payload` | JSONB | |
| `created_at` | TIMESTAMPTZ | SSE cursor primary key |

Index: `(session_id, created_at ASC, id ASC)` — supports SSE cursor queries.

---

## Hasura RLS

All three tables apply:
- `user` role: `{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}`
- `nself_admin` role: `{}` (no filter — admin sees all)

---

## Security Model

| Concern | Mitigation |
|---|---|
| Daemon reachability disclosure | `/health` is plugin-local only; daemon requires auth |
| Missing tenant header | requireAuth checks both `Authorization` AND `X-Hasura-Tenant-Id` independently |
| Cross-tenant event access | `WHERE tenant_id = $tenantId` on all queries + Hasura RLS |
| Session ownership on SSE | Session existence + tenant ownership verified before stream opens |
| Event loss on reconnect | `(created_at, id)` cursor — no UUID comparison |

---

## Development

```bash
cd plugins-pro/paid/plugin-clawde

# Run tests
go test ./...

# Build
go build ./...

# Check for issues
go vet ./...
```

---

## Docker

```bash
# Build
docker build -t nself/plugin-clawde:latest .

# Health check
curl http://localhost:3847/health
```

---

## Changelog

### v1.0.0 (P4-E4-W1-S02-T06)

- Initial implementation
- Session lifecycle management
- Daemon health proxy (auth-gated)
- SSE event streaming with `(created_at, id)` cursor
- Tenant RLS on all three tables
- ClawDE bundle membership

---

## Related

- [ClawDE bundle](https://nself.org/products/clawde)
- [plugin-clawde-pty](plugin-plugin-clawde-pty.md) — PTY bridge (separate plugin)
- [clawde/ repo](https://github.com/nself-org/clawde) — Flutter desktop app
