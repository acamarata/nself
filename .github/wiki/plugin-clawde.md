# plugin-clawde

**Bundle:** ClawDE · **Port:** 3847 · **Tier:** pro · **License:** requires_license=true

ClawDE daemon integration backend for the nSelf platform. Bridges the ClawDE AI development environment with the nSelf backend — providing daemon session management, tool registration, heartbeat keepalive, and tenant-isolated state persistence.

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Installation](#installation)
4. [Configuration](#configuration)
5. [API Reference](#api-reference)
   - [Health](#health)
   - [Daemon Status](#daemon-status)
   - [Session Lifecycle](#session-lifecycle)
   - [Heartbeat](#heartbeat)
   - [Tool Registration](#tool-registration)
   - [Events](#events)
   - [Admin: Expire Sweep](#admin-expire-sweep)
6. [Database Schema](#database-schema)
7. [Hasura RLS](#hasura-rls)
8. [License Gate](#license-gate)
9. [Troubleshooting](#troubleshooting)

---

## Overview

`plugin-clawde` is the nSelf backend plugin that ClawDE daemons connect to. It serves three primary functions:

1. **Session lifecycle** — Daemons register a session on connect; the session tracks status, daemon address, and user context. Sessions expire 60 seconds after the last heartbeat.
2. **Tool registration** — Daemons register the tools they expose (with sanitized JSON schemas). Clients query registered tools per session.
3. **Heartbeat keepalive** — Daemons POST to `/sessions/{id}/heartbeat` every ≤30 s to keep their session active.

All data is tenant-isolated via `tenant_id` enforced at both the application layer and Hasura row-level security.

---

## Architecture

```
ClawDE Desktop App
       │
       ▼
ClawDE Daemon (local process)
       │  POST /sessions          — register session
       │  POST /sessions/{id}/heartbeat  — keepalive
       │  POST /sessions/{id}/tools      — register tools
       ▼
plugin-clawde (port 3847)
       │
       ▼
PostgreSQL (np_clawde_sessions, np_clawde_tool_registrations,
            np_clawde_heartbeats, np_clawde_events, np_clawde_daemon_status)
       │
       ▼
Hasura GraphQL (row-filter: tenant_id = X-Hasura-Tenant-Id)
```

Sessions expire if no heartbeat is received for 60 seconds. The `/sessions/expire` admin endpoint or a cron job triggers the sweep.

---

## Installation

```bash
nself plugin install plugin-clawde
```

Requires a valid ClawDE bundle or ɳSelf+ license key. Purchase at [nself.org/products/clawde](https://nself.org/products/clawde).

Verify installation:

```bash
nself plugin list | grep clawde
nself plugin status plugin-clawde
```

---

## Configuration

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `CLAWDE_PLUGIN_PORT` | No | `3847` | HTTP listen port |
| `CLAWDE_PLUGIN_HOST` | No | `0.0.0.0` | HTTP listen host |
| `CLAWDE_DAEMON_URL` | No | — | Daemon base URL for health probing |
| `CLAWDE_DAEMON_TOKEN` | No | — | Auth token for daemon health probe |
| `CLAWDE_DAEMON_HEALTH_INTERVAL` | No | `30s` | Daemon health probe interval |
| `CLAWDE_SESSION_TTL_HOURS` | No | `24` | Max session age (hours) |
| `CLAWDE_EVENT_BUFFER_SIZE` | No | `1000` | Max in-flight events buffered |
| `CLAWDE_MAX_SESSIONS_PER_TENANT` | No | `10` | Max concurrent active sessions per tenant |
| `LOG_LEVEL` | No | `info` | Log level (`debug`/`info`/`warn`/`error`) |

---

## API Reference

All endpoints except `/health` require `X-Hasura-Tenant-Id` header (set by Hasura JWT middleware upstream).

### Health

```
GET /health
```

Returns 200 + `{"status":"ok"}`. No auth required. Used by Docker HEALTHCHECK.

---

### Daemon Status

```
GET /daemon/health   — Auth: bearer
GET /daemon/status   — Auth: bearer
```

Probes the configured ClawDE daemon URL and returns its health and status. Requires `CLAWDE_DAEMON_URL` to be set.

---

### Session Lifecycle

#### Register a session

```
POST /sessions
Content-Type: application/json
X-Hasura-Tenant-Id: <tenant_uuid>

{
  "user_id": "user-123",
  "name": "my-session"         // optional; auto-generated if omitted
}
```

Response `201`:

```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "user_id": "user-123",
  "name": "my-session",
  "status": "active",
  "started_at": "2026-06-22T...",
  "created_at": "2026-06-22T..."
}
```

Max sessions per tenant: 10 (configurable via `CLAWDE_MAX_SESSIONS_PER_TENANT`). Returns 409 if exceeded.

#### List sessions

```
GET /sessions
X-Hasura-Tenant-Id: <tenant_uuid>
```

#### Get a session

```
GET /sessions/{id}
X-Hasura-Tenant-Id: <tenant_uuid>
```

Returns 404 if session not found for this tenant.

#### Close a session

```
DELETE /sessions/{id}
X-Hasura-Tenant-Id: <tenant_uuid>
```

Marks session `closed`. Returns 404 if not found.

---

### Heartbeat

Daemons must send a heartbeat every ≤30 seconds. Sessions without a heartbeat for 60 seconds are marked `idle` on the next expire sweep.

```
POST /sessions/{id}/heartbeat
X-Hasura-Tenant-Id: <tenant_uuid>
```

Response `200`:

```json
{
  "session_id": "uuid",
  "last_heartbeat": "2026-06-22T...",
  "alive": true
}
```

Returns 409 if the session is already `closed`.

---

### Tool Registration

Daemons register the tools they expose. Tool schemas are sanitized before storage (forbidden fields stripped: `$schema`, `$ref`, `x-code`, `x-eval`, `x-exec`, `x-script`).

#### Register / update a tool

```
POST /sessions/{id}/tools
Content-Type: application/json
X-Hasura-Tenant-Id: <tenant_uuid>

{
  "tool_name": "read_file",
  "schema": {
    "description": "Read a file from the workspace",
    "parameters": {
      "type": "object",
      "properties": {
        "path": { "type": "string" }
      },
      "required": ["path"]
    }
  }
}
```

Response `201` (or upsert on duplicate tool_name within the same session).

#### List tools for a session

```
GET /sessions/{id}/tools
X-Hasura-Tenant-Id: <tenant_uuid>
```

Response `200`:

```json
{
  "tools": [...],
  "count": 3
}
```

#### Delete a tool

```
DELETE /sessions/{id}/tools/{toolName}
X-Hasura-Tenant-Id: <tenant_uuid>
```

Returns 204 on success, 404 if not found.

---

### Events

```
GET  /sessions/{id}/events   — list events for a session
POST /sessions/{id}/events   — append an event
```

Events are append-only and tenant-scoped. Useful for debugging and audit.

---

### Admin: Expire Sweep

```
POST /sessions/expire
X-Hasura-Tenant-Id: <tenant_uuid>
```

Marks all active sessions with `last_heartbeat` older than 60 s as `idle`. Also expires sessions with no heartbeat row that are older than 60 s. Returns counts of rows affected.

Intended for admin/cron use. Wire to a `nself cron` job or call from a health monitor.

---

## Database Schema

Five tables, all prefixed `np_clawde_`:

| Table | Key Columns | Purpose |
|---|---|---|
| `np_clawde_sessions` | `id, tenant_id, user_id, status, daemon_addr, source_account_id` | Active dev sessions |
| `np_clawde_daemon_status` | `id, tenant_id, daemon_addr, is_healthy, last_probe, source_account_id` | Daemon health state |
| `np_clawde_events` | `id, tenant_id, session_id, event_type, payload, source_account_id` | Session audit events |
| `np_clawde_tool_registrations` | `id, tenant_id, session_id, tool_name, schema, source_account_id` | Tools registered by daemon |
| `np_clawde_heartbeats` | `id, tenant_id, session_id, last_heartbeat, source_account_id` | Heartbeat keepalive state |

All tables carry `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation.

Migrations: `paid/plugin-clawde/migrations/001_init.sql` + `002_tool_registrations_heartbeats.sql`.

---

## Hasura RLS

All `np_clawde_*` tables enforce `{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}` on all roles (select/insert/update/delete). Metadata: `paid/plugin-clawde/hasura/metadata.yaml`.

---

## License Gate

plugin-clawde requires a valid **ClawDE bundle** or **ɳSelf+** license key.

- Install validates via `ping.nself.org/license/validate`
- Invalid or missing key → install fails with purchase URL
- License validation uses FAIL-OPEN with 7-day TTL (if ping_api is unreachable, last valid token is honoured for 7 days)

Purchase or manage licenses: `nself license` or [nself.org/products/clawde](https://nself.org/products/clawde).

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `409 max active sessions per tenant reached` | Too many concurrent sessions | Close stale sessions via `DELETE /sessions/{id}` or increase `CLAWDE_MAX_SESSIONS_PER_TENANT` |
| Session keeps going `idle` | Daemon heartbeat interval too long | Set daemon heartbeat ≤30 s; plugin expires after 60 s silence |
| `404 session not found` on heartbeat | Session closed or wrong tenant header | Verify `X-Hasura-Tenant-Id` matches session's tenant |
| `/health` returns 503 | DB connection down | Check `DATABASE_URL` and PostgreSQL availability |
| Tool schema rejected | Forbidden fields in schema | Remove `$schema`, `$ref`, `x-code`, `x-eval`, `x-exec`, `x-script` from schema |

For further help: `nself doctor --plugin plugin-clawde` or [nself.org/support](https://nself.org/support).
