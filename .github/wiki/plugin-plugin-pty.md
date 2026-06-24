# plugin-pty — PTY Bridge Plugin

**Bundle:** ClawDE | **Port:** 9100 | **License:** requires_license=true | **Version:** 1.0.0

## Overview

`plugin-pty` is the PTY bridge plugin for ɳSelf. It manages pseudo-terminal (PTY) sessions between the ClawDE daemon and the nSelf backend, enabling real-time terminal I/O streaming over WebSocket.

The plugin is **scoped exclusively to the Claude Max binary**. It does not provide arbitrary shell access. The exec target is set via the `PTY_CLAUDE_BIN` environment variable and validated at startup — clients cannot supply or override it at session-creation time.

## Security Model

- **Exec target is config-only.** `PTY_CLAUDE_BIN` must point to the Claude Max binary. It is an absolute path validated at startup. No user-supplied paths are accepted.
- **No shell injection.** The child process receives a minimal, safe environment: `HOME`, `TERM`, `LANG`, `PATH`, and `CLAUDE_*` prefixed vars only. `DATABASE_URL`, `INTERNAL_SECRET`, and all other host secrets are **never** passed to the child process.
- **One bridge per session.** A WebSocket connection atomically claims `status='bridging'` from `status='active'` with a database-level UPDATE. Concurrent connections on the same session are rejected with HTTP 409.
- **Tenant isolation via Hasura JWT.** The `source_account_id` is read exclusively from the `X-Hasura-Source-Account-Id` header, set by the Hasura JWT middleware. User-supplied headers (`X-Tenant-Id`) are never trusted.
- **Full audit log.** Every session open, close, error, and resize event is written to `np_pty_audit`.

## Installation

```bash
nself plugin install plugin-pty
```

Requires an active ClawDE bundle license.

## Configuration

| Environment Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `PTY_CLAUDE_BIN` | Yes | — | Absolute path to the Claude Max binary |
| `PORT` | No | `9100` | HTTP/WS listen port |
| `INTERNAL_SECRET` | No | — | Service-to-service auth token |
| `PTY_MAX_SESSIONS` | No | `50` | Max concurrent active sessions |
| `PTY_SESSION_TIMEOUT_SECONDS` | No | `3600` | Session idle timeout (0 = disabled) |

## Database Schema

### `np_pty_sessions`

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary key |
| `source_account_id` | TEXT | Multi-app isolation key (default: `'primary'`) |
| `user_id` | TEXT | Authenticated user identifier |
| `status` | TEXT | `active` / `bridging` / `closed` / `error` |
| `claude_bin_path` | TEXT | Binary path used for this session |
| `pid` | INTEGER | Child process PID |
| `rows` | SMALLINT | Terminal rows |
| `cols` | SMALLINT | Terminal columns |
| `started_at` | TIMESTAMPTZ | Session start time |
| `closed_at` | TIMESTAMPTZ | Session end time (nullable) |
| `error_msg` | TEXT | Error detail (nullable) |

### `np_pty_audit`

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary key |
| `session_id` | UUID | FK → np_pty_sessions |
| `source_account_id` | TEXT | Isolation key (redundant for fast queries) |
| `event` | TEXT | `opened` / `closed` / `error` / `resize` |
| `detail` | JSONB | Event-specific payload |
| `created_at` | TIMESTAMPTZ | Event timestamp |

## WebSocket API

Connect to: `ws://<host>:9100/ws/sessions/{session_id}/bridge`

### Client → Server Messages

**Input (text/data to PTY stdin):**
```json
{"type": "input", "data": "ls -la\n"}
```

**Resize terminal:**
```json
{"type": "resize", "rows": 40, "cols": 120}
```

### Server → Client Messages

**PTY stdout output:**
```json
{"type": "output", "data": "total 32\ndrwxr-xr-x ..."}
```

**Error:**
```json
{"type": "error", "data": "pty start failed"}
```

## HTTP API

### `GET /health`

Returns `200 OK` with:
```json
{"status": "ok", "plugin": "plugin-pty", "version": "1.0.0"}
```

### `GET /sessions`

Returns all PTY sessions for the authenticated source account. Optional `?status=active` filter.

Response:
```json
[
  {
    "id": "uuid",
    "source_account_id": "primary",
    "user_id": "user-123",
    "status": "active",
    "rows": 24,
    "cols": 80,
    "started_at": "2026-06-24T10:00:00Z"
  }
]
```

### `POST /sessions/{id}/close`

Closes an active or bridging session. Returns `204 No Content` on success.

## Session Lifecycle

```
Client connects (WS) → session status='bridging' (atomic DB claim)
    → PTY spawned (claude binary, safe env)
    → I/O pumped: WS ↔ PTY
    → Client disconnects → PTY killed → status='closed'
```

## Hasura RLS

Both tables use the Multi-App Isolation pattern:

```yaml
filter:
  source_account_id:
    _eq: X-Hasura-Source-Account-Id
```

All read/write operations through Hasura are scoped to the caller's `source_account_id`. There is no cross-account data visibility.

## Metrics

`GET /metrics` — Prometheus-compatible (coming in v1.1.0).

## Changelog

| Version | Date | Notes |
|---|---|---|
| 1.0.0 | 2026-06-24 | Initial release — WebSocket PTY bridge, session lifecycle, audit log, Hasura RLS |

## Related

- `nself-ai-cc` (port 3760) — Claude Code session relay (same bundle)
- `nself-ai-gateway` (port 3761) — AI provider routing (same bundle)
- ClawDE bundle: `clawde.io`
