# plugin-clawde

**ClawDE daemon integration backend** — nSelf paid plugin, ClawDE bundle.

| Field | Value |
|-------|-------|
| **Name** | `plugin-clawde` |
| **Bundle** | ClawDE |
| **Tier** | `pro` (`requires_license: true`) |
| **Port** | `3847` |
| **Language** | Go |
| **Version** | v1.0.0 |
| **Min nSelf** | v1.1.1 |
| **Category** | integrations |
| **Docker image** | `nself/plugin-clawde:latest` |

---

## Overview

`plugin-clawde` is the backend counterpart to the [ClawDE](https://clawde.io) AI development environment desktop and mobile app. It runs as a standard nSelf plugin container and provides:

- **Daemon health probing** — polls the ClawDE daemon via HTTP and persists health snapshots per tenant
- **Session lifecycle management** — creates, tracks, and closes ClawDE dev sessions with full tenant isolation
- **Event streaming** — append-only event log per session; supports Server-Sent Events (SSE) for real-time delivery to connected clients
- **Tenant RLS enforcement** — all DB tables carry `tenant_id`; Hasura row-filters prevent cross-tenant data exposure at the GraphQL layer

---

## Installation

```bash
# Requires ClawDE bundle or ɳSelf+ license
nself plugin install plugin-clawde

# Verify
nself plugin status plugin-clawde
```

---

## Configuration

Set the following environment variables in your nSelf `.env` or via `nself config set`:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | **Yes** | — | Postgres DSN (same as nSelf core) |
| `CLAWDE_PLUGIN_PORT` | No | `3847` | HTTP bind port |
| `CLAWDE_PLUGIN_HOST` | No | `0.0.0.0` | HTTP bind host |
| `CLAWDE_DAEMON_URL` | No | — | Base URL of the running ClawDE daemon process |
| `CLAWDE_DAEMON_TOKEN` | No | — | Bearer token for daemon authentication |
| `CLAWDE_DAEMON_HEALTH_INTERVAL` | No | `30` | Seconds between health probes |
| `CLAWDE_DAEMON_CONNECT_TIMEOUT` | No | `5` | Probe timeout in seconds |
| `CLAWDE_SESSION_TTL_HOURS` | No | `24` | Auto-close idle sessions after N hours |
| `CLAWDE_MAX_SESSIONS_PER_TENANT` | No | `10` | Max concurrent active sessions per tenant |
| `CLAWDE_EVENT_BUFFER_SIZE` | No | `1000` | In-memory SSE event buffer size |
| `LOG_LEVEL` | No | `info` | Log verbosity (`debug`, `info`, `warn`, `error`) |

### Minimal configuration

```env
DATABASE_URL=postgres://nself:nself@localhost:5432/nself
CLAWDE_DAEMON_URL=http://localhost:4200
CLAWDE_DAEMON_TOKEN=your-daemon-auth-token
```

---

## Database Tables

All tables use the `np_clawde_*` prefix per nSelf plugin convention. Every table includes a `tenant_id UUID NOT NULL` column enforced by Hasura RLS.

### `np_clawde_sessions`

Per-tenant ClawDE session state.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | Session identifier |
| `tenant_id` | UUID | Hasura RLS column |
| `user_id` | TEXT | User who owns this session |
| `name` | TEXT | Human-readable session label |
| `status` | TEXT | `active` / `idle` / `closed` / `error` |
| `metadata` | JSONB | Arbitrary session context |
| `daemon_addr` | TEXT | Assigned daemon endpoint (nullable) |
| `started_at` | TIMESTAMPTZ | Session open time |
| `closed_at` | TIMESTAMPTZ | Session close time (nullable) |
| `created_at` | TIMESTAMPTZ | Record creation |
| `updated_at` | TIMESTAMPTZ | Last update |

### `np_clawde_daemon_status`

Daemon health snapshots per tenant.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `tenant_id` | UUID | Hasura RLS column |
| `daemon_addr` | TEXT | Daemon endpoint probed |
| `is_healthy` | BOOLEAN | Last probe result |
| `version` | TEXT | Daemon version (nullable) |
| `last_probe` | TIMESTAMPTZ | When last probed |
| `error_msg` | TEXT | Last error (nullable) |

### `np_clawde_events`

Append-only session event log.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `tenant_id` | UUID | Hasura RLS column |
| `session_id` | UUID | FK → `np_clawde_sessions.id` ON DELETE CASCADE |
| `event_type` | TEXT | e.g. `daemon.connected`, `file.opened`, `command.run` |
| `payload` | JSONB | Event-specific data |
| `created_at` | TIMESTAMPTZ | Immutable append timestamp |

---

## HTTP API

All routes except `/health` require a valid bearer token. The `X-Hasura-Tenant-Id` header is set automatically by the Hasura JWT middleware in front of this plugin.

### GET `/health`

Unauthenticated liveness check.

```bash
curl http://localhost:3847/health
# {"status":"ok","service":"plugin-clawde","time":"2026-06-22T10:00:00Z"}
```

### GET `/daemon/health`

Live probe of the configured ClawDE daemon.

```bash
curl http://localhost:3847/daemon/health \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Hasura-Tenant-Id: $TENANT_ID"
# {"daemon_healthy":true,"version":"1.2.0","checked_at":"..."}
```

Returns `503` if daemon is unreachable or unhealthy.

### GET `/daemon/status`

Returns the last cached daemon health record from the DB (falls back to live probe if no record found).

### POST `/sessions`

Create a new ClawDE session.

**Request body:**

```json
{
  "user_id": "user-123",
  "name": "my-dev-session"
}
```

**Response:** `201 Created`

```json
{
  "id": "uuid-...",
  "tenant_id": "uuid-...",
  "user_id": "user-123",
  "name": "my-dev-session",
  "status": "active",
  "started_at": "2026-06-22T10:00:00Z",
  "created_at": "2026-06-22T10:00:00Z"
}
```

Returns `409 Conflict` if `CLAWDE_MAX_SESSIONS_PER_TENANT` is reached.

### GET `/sessions`

List all sessions for the requesting tenant.

```bash
curl http://localhost:3847/sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Hasura-Tenant-Id: $TENANT_ID"
# {"sessions":[...], "count":3}
```

### GET `/sessions/{id}`

Get a single session by ID. Returns `404` if the session does not belong to the requesting tenant.

### DELETE `/sessions/{id}`

Close a session. Sets `status=closed` and `closed_at=now()`. Returns `404` if not found or already closed.

### GET `/sessions/{id}/events`

List events for a session.

```bash
curl http://localhost:3847/sessions/$SESSION_ID/events \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Hasura-Tenant-Id: $TENANT_ID"
# {"events":[...], "count":12}
```

**SSE streaming** — append `?stream=true` for real-time event delivery:

```bash
curl -N "http://localhost:3847/sessions/$SESSION_ID/events?stream=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Hasura-Tenant-Id: $TENANT_ID"
```

### POST `/sessions/{id}/events`

Append an event to a session.

**Request body:**

```json
{
  "event_type": "file.opened",
  "payload": {"path": "/src/main.go", "line": 42}
}
```

---

## Hasura RLS

All three `np_clawde_*` tables are protected by Hasura row-level security. The row filter applied to all operations on the `user` role is:

```json
{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}
```

This is declared in `hasura/metadata.yaml` and applied via:

```bash
nself hasura apply --plugin plugin-clawde
```

Even if the HTTP API were bypassed, the GraphQL layer enforces isolation — a tenant can never access another tenant's sessions, daemon status records, or events.

---

## Session Lifecycle

```
POST /sessions → status=active
  │
  ├── ClawDE daemon connects
  │   └── POST /sessions/{id}/events {"event_type":"daemon.connected"}
  │
  ├── Development work events
  │   └── POST /sessions/{id}/events {"event_type":"file.opened", ...}
  │
  ├── Idle (no events for TTL hours)
  │   └── background job sets status=idle
  │
  └── DELETE /sessions/{id}
      └── status=closed, closed_at=now()
```

---

## Security

- All endpoints except `/health` require bearer authentication
- `X-Hasura-Tenant-Id` is required for all session/event operations — absence returns `400 Bad Request`
- Tenant isolation: all DB queries include `AND tenant_id=$tenantID`; Hasura adds a second enforcement layer
- `CLAWDE_DAEMON_TOKEN` is never returned in any API response
- Events are append-only — no UPDATE or DELETE on `np_clawde_events`
- Max sessions per tenant prevents resource exhaustion (`CLAWDE_MAX_SESSIONS_PER_TENANT`, default 10)

---

## Docker

```bash
# Pull
docker pull nself/plugin-clawde:latest

# Run manually (nself manages this in production)
docker run -d \
  -e DATABASE_URL=postgres://nself:nself@db:5432/nself \
  -e CLAWDE_DAEMON_URL=http://clawd:4200 \
  -e CLAWDE_DAEMON_TOKEN=secret \
  -p 3847:3847 \
  nself/plugin-clawde:latest
```

The container exposes port `3847` and uses `HEALTHCHECK` via `wget /health` every 30 seconds.

---

## Development

```bash
cd plugins-pro/paid/plugin-clawde

# Run tests
go test ./...

# Build
go build -o plugin-clawde ./cmd

# Start locally (requires Postgres)
DATABASE_URL=postgres://nself:nself@localhost:5432/nself ./plugin-clawde
```

---

## Related

| Resource | Link |
|----------|------|
| ClawDE app repo | `~/Sites/nself/clawde/` |
| plugin-pty | `plugins-pro/paid/plugin-pty/` — PTY bridge for terminal sessions |
| plugin-llm-gateway | `plugins-pro/paid/plugin-llm-gateway/` — AI gateway for ClawDE |
| plugin-retrieval | `plugins-pro/paid/plugin-retrieval/` — Hybrid vector+BM25 retrieval |
| SPORT F04 | `.claude/docs/sport/F04-PLUGIN-INVENTORY.md` — plugin inventory |
| SPORT F10 | `.claude/docs/sport/F10-PORT-REGISTRY.md` — port registry (3847) |
| Ticket | P4-E4-W1-S02-T06 |

---

*plugin-clawde v1.0.0 — nSelf ClawDE bundle — Source-Available*
