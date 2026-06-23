# plugin-pty

> PTY bridge management for Claude Max sessions within ClawDE.

**Bundle:** ClawDE | **Port:** 9100 | **License:** Pro (requires_license: true)

---

## Overview

plugin-pty manages pseudo-terminal (PTY) bridge lifecycles so ClawDE can relay I/O between its UI and a running Claude Max (or any operator-configured) process. Each WebSocket connection corresponds to one PTY session, scoped to a tenant.

Key guarantees:

- One exec binary per deploy, set by the operator — clients cannot specify which binary runs
- Per-tenant concurrency cap (default 5 sessions; configurable via env var)
- Cross-tenant access returns 403 at the WebSocket upgrade step
- Append-only audit log of all session lifecycle events

---

## Install

```sh
nself plugin install plugin-pty
```

Requires a valid ClawDE bundle or ɳSelf+ license. The CLI validates the license against `ping.nself.org` before downloading.

---

## Configuration

Set these environment variables (via `nself env set` or `nself.yaml`):

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `PTY_EXEC_PATH` | Yes | — | Absolute path to PTY exec binary |
| `PTY_PORT` / `PORT` | No | `9100` | HTTP listen port |
| `PTY_HOST` / `HOST` | No | `0.0.0.0` | Bind address |
| `PTY_SESSION_TTL_SECONDS` | No | `3600` | Max PTY session lifetime (seconds) |
| `PTY_MAX_SESSIONS_PER_TENANT` | No | `5` | Max concurrent PTY sessions per tenant |
| `PTY_READ_BUFFER_SIZE` | No | `4096` | I/O read buffer (bytes) |
| `PTY_WRITE_BUFFER_SIZE` | No | `4096` | I/O write buffer (bytes) |

`PTY_EXEC_PATH` must be an absolute path to a binary that exists on the host. It is validated at startup — the service will not start if the binary is missing or the path is relative.

---

## API Reference

### GET /health

No authentication required. Returns `{"status":"ok"}` when the DB is reachable, `{"status":"degraded","error":"..."}` (HTTP 503) otherwise.

Used by Docker HEALTHCHECK and `nself doctor`.

---

### POST /sessions

**Auth:** Bearer token (tenant JWT with `X-Hasura-Tenant-Id` claim)

Spawns a new PTY session for the authenticated tenant.

Returns **429 Too Many Requests** if `PTY_MAX_SESSIONS_PER_TENANT` active sessions already exist for this tenant.

**Response 201:**

```json
{
  "id": "3f8a1b2c-4d5e-6f7a-8b9c-0d1e2f3a4b5c",
  "tenant_id": "a1b2c3d4-...",
  "session_token": "unique-uuid-token",
  "status": "active",
  "created_at": "2026-01-01T00:00:00Z"
}
```

---

### GET /sessions

**Auth:** Bearer

List all sessions for the authenticated tenant (most recent first, up to 100).

---

### GET /sessions/{id}

**Auth:** Bearer

Get a single session. Returns 404 if the session does not belong to the authenticated tenant.

---

### DELETE /sessions/{id}

**Auth:** Bearer

Closes the session (status → `closed`), writes a `session_closed` audit entry, and hard-deletes the row.

---

### WS /sessions/{id}/ws

**Auth:** Bearer (header `Authorization: Bearer <token>` or `X-Hasura-Tenant-Id`)

Upgrades the connection to WebSocket and starts the PTY bridge.

**Behavior:**

- Validates that the session exists and belongs to the authenticated tenant (403 otherwise)
- Spawns the binary at `PTY_EXEC_PATH`
- Client frames (binary or text) are forwarded verbatim as stdin
- Process stdout and stderr are forwarded to the client as binary frames
- Session closes automatically when: (a) the PTY process exits, (b) the client disconnects, or (c) `PTY_SESSION_TTL_SECONDS` elapses

---

### GET /audit

**Auth:** Bearer

Returns recent audit log entries for the authenticated tenant. Supports `?limit=N` (max 500, default 100).

**Response:**

```json
{
  "events": [
    {
      "id": "...",
      "tenant_id": "...",
      "session_id": "...",
      "event_type": "session_created",
      "details": {},
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

**Event types:** `session_created` · `session_closed` · `session_error` · `io_error` · `auth_denied` · `injection_blocked`

---

## Security

### Exec Path Isolation

The binary executed by the PTY is always `PTY_EXEC_PATH`. This value comes exclusively from the server environment (set by the nSelf operator at install time). No WebSocket message, URL parameter, query string, or HTTP header can change which binary runs.

### Tenant Isolation

Every database query, WebSocket upgrade, and session lookup is scoped to the tenant ID extracted from the JWT claim `X-Hasura-Tenant-Id`. Attempting to access another tenant's session returns 403.

### Resource Limits

`PTY_MAX_SESSIONS_PER_TENANT` (default 5) caps concurrent active sessions. Requests that would exceed the limit receive HTTP 429 with the limit value in the response body.

### Audit Log Immutability

The `np_pty_audit_log` table has a PostgreSQL trigger (`np_pty_audit_log_no_update`) that raises an exception on any UPDATE or DELETE. Audit records cannot be tampered with at the application level.

---

## Database Tables

### `np_pty_sessions`

Tracks active and historical PTY bridge sessions.

```sql
CREATE TABLE np_pty_sessions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    source_account_id TEXT NOT NULL DEFAULT 'primary',
    session_token     TEXT NOT NULL UNIQUE,
    status            TEXT NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active', 'closed', 'error')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at         TIMESTAMPTZ,
    error_msg         TEXT,
    metadata          JSONB NOT NULL DEFAULT '{}'
);
```

### `np_pty_audit_log`

Append-only lifecycle event log.

```sql
CREATE TABLE np_pty_audit_log (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    source_account_id TEXT NOT NULL DEFAULT 'primary',
    session_id        UUID REFERENCES np_pty_sessions(id) ON DELETE SET NULL,
    event_type        TEXT NOT NULL CHECK (...),
    details           JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## Hasura Permissions

Hasura row-level filters on both tables use `{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}` for all roles. The audit log has no update or delete permissions.

---

## Docker

```sh
docker pull nself/plugin-pty:latest
```

Supported: `linux/amd64` · `linux/arm64` · `darwin/arm64`

Health check runs every 30 seconds on port 9100 (`/health`).

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Service won't start | `PTY_EXEC_PATH` missing or not absolute | Set correct absolute path in env |
| HTTP 401 on all requests | Missing `X-Hasura-Tenant-Id` header | Ensure JWT carries tenant claim |
| HTTP 429 on POST /sessions | Per-tenant limit reached | Increase `PTY_MAX_SESSIONS_PER_TENANT` or delete old sessions |
| WS 403 | Session belongs to different tenant | Use matching tenant credentials |
| `/health` returns degraded | DB unreachable | Check `DATABASE_URL` and Postgres connectivity |

---

## Changelog

| Version | Change |
|---------|--------|
| 1.0.0 | Initial release — PTY bridge, REST + WS API, tenant isolation, audit log |

---

## See Also

- [plugin-clawde](plugin-clawde.md) — ClawDE daemon integration
- [nself plugin install](commands/cmd-plugin.md)
- [nSelf License](https://nself.org/products/clawde)
