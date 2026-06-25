# plugin-pty

PTY bridge plugin for ClawDE AI sessions. Spawns and manages pseudo-terminal processes with
WebSocket I/O relay and per-tenant resource limits.

**Port:** 9100 | **Bundle:** ClawDE | **License:** Pro (requires_license=true)

---

## Overview

plugin-pty bridges the ClawDE desktop app with terminal sessions on the nSelf backend.
The ClawDE client creates a PTY session via REST, receives a WebSocket URL, and connects
for bidirectional I/O. Each session runs an isolated shell process.

Key properties:
- Per-tenant session limit (default 5) — 429 on exceed
- No SSRF surface (no outbound HTTP calls)
- Lifecycle events audited in `np_pty_audit_log`
- All data scoped by `source_account_id` (Multi-App Isolation Convention)

---

## Installation

```bash
nself plugin install plugin-pty
```

Requires nSelf v1.1.1+ and a ClawDE bundle license.

---

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NSELF_DB_URL` | Yes | — | PostgreSQL connection string |
| `NSELF_LICENSE_KEY` | Yes | — | nSelf license key (pro) |
| `PTY_MAX_PER_TENANT` | No | `5` | Max concurrent PTY sessions per tenant |
| `PTY_SESSION_TIMEOUT_SECS` | No | `3600` | Session idle timeout in seconds |
| `PTY_PORT` | No | `9100` | HTTP server port |

---

## API Reference

### GET /health

Returns `{"status":"ok"}` (200) when DB is reachable.

### POST /sessions

Spawn a new PTY session.

**Request:**
```json
{ "client_id": "clawde-daemon-xyz", "cols": 120, "rows": 30 }
```

**Response (201):**
```json
{
  "session_id": "550e8400-...",
  "ws_url": "/sessions/550e8400-.../ws",
  "status": "active"
}
```

**Errors:**
- `400` — client_id missing
- `403` — invalid license
- `429` — max concurrent sessions exceeded

### DELETE /sessions/{id}

Close a session. Terminates the underlying PTY process.

**Response:** 204 No Content

### GET /sessions/{id}/ws

Upgrade to WebSocket for bidirectional PTY I/O.

- **Server → Client:** binary frames containing PTY stdout/stderr
- **Client → Server:** binary frames containing PTY stdin

Requires `X-Hasura-Source-Account-Id` header matching the session owner. Returns `403` if
source account does not match.

---

## Database Schema

### np_pty_sessions

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Session identifier |
| `source_account_id` | TEXT | Account scope (Convention Wall) |
| `client_id` | TEXT | ClawDE daemon identifier |
| `status` | TEXT | active / closed / expired |
| `cols` | INTEGER | Terminal width |
| `rows` | INTEGER | Terminal height |
| `started_at` | TIMESTAMPTZ | Session start time |
| `ended_at` | TIMESTAMPTZ | Session end time (null if active) |

### np_pty_audit_log

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Audit entry ID |
| `source_account_id` | TEXT | Account scope |
| `session_id` | UUID | Related session |
| `event_type` | TEXT | spawn / close / resize / error |
| `detail` | TEXT | Event details |
| `created_at` | TIMESTAMPTZ | Event timestamp |

---

## Security Model

- **Session isolation:** PTY processes are per-session; no cross-session I/O access
- **Tenant isolation:** `source_account_id` scopes all DB rows; Hasura row-filter enforces
- **Rate limiting:** max sessions per tenant prevents resource exhaustion
- **License gate:** `NSELF_LICENSE_KEY` required at startup
- **No SSRF:** plugin makes no outbound HTTP calls

---

## Integration with ClawDE

ClawDE desktop connects to this plugin at `http://localhost:9100` (default). Session lifecycle
is managed by `clawde/lib/backend/pty_client.dart`. The WebSocket connection is maintained
for the duration of each terminal tab.

To configure a custom backend address in ClawDE, set `PTY_RELAY_URL` in the ClawDE plugin
settings or `NSELF_CLAWDE_PTY_URL` environment variable.
