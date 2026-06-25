# plugin-pty — PTY Bridge (ClawDE Bundle)

## Overview

`plugin-pty` provides a WebSocket PTY bridge for ClawDE AI sessions. It connects the
ClawDE daemon to the nSelf backend via a persistent pseudo-terminal, enabling full
terminal-native AI interaction with session tracking and security audit logging.

**Port**: 9100 | **Bundle**: ClawDE | **Tier**: Paid | **License**: Required

## Installation

```bash
nself plugin install plugin-pty
```

Set license via environment:

```bash
export NSELF_LICENSE_KEY=your-clawde-license-key
```

## Architecture

```
ClawDE Daemon ─── WebSocket (ws://:9100/ws) ─── plugin-pty ─── PTY ─── ClawDE Binary
                                                     │
                                            Postgres (np_pty_*)
```

The PTY exec target is configured once at startup via `PTY_CLAWDE_BINARY_PATH`. This
path is never overridable per WebSocket request — preventing shell injection.

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | Postgres connection string |
| `NSELF_LICENSE_KEY` | Yes | — | ClawDE or ɳSelf+ license key |
| `PTY_CLAWDE_BINARY_PATH` | No | `/usr/local/bin/clawde` | Fixed ClawDE binary path |
| `PTY_MAX_SESSIONS` | No | `50` | Concurrent session limit |
| `PTY_SESSION_TIMEOUT_SECONDS` | No | `3600` | Idle timeout |
| `PORT` | No | `9100` | HTTP/WS listen port |

## API Reference

### Health Check

```
GET /health
→ {"status":"ok","plugin":"plugin-pty","port":9100}
```

### Open PTY Session

```
GET /ws?client_id=<id>
Upgrade: websocket
X-Nself-License: <key>
X-Source-Account-ID: <account-id>
```

After upgrade, the connection is a bidirectional byte stream to the PTY process.
Send keyboard input as raw bytes; receive terminal output as raw bytes.

### List Sessions

```
GET /sessions
X-Nself-License: <key>
X-Source-Account-ID: <account-id>

→ [{"id":"...","client_id":"...","status":"open","cols":80,"rows":24,...}]
```

## Database Schema

### `np_pty_sessions`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `source_account_id` | TEXT | Tenant isolation (Convention Wall) |
| `client_id` | TEXT | ClawDE daemon client identifier |
| `pid` | INTEGER | OS process ID of PTY process |
| `status` | VARCHAR | `open` / `closed` / `error` |
| `cols`, `rows` | INTEGER | Terminal dimensions |
| `opened_at` | TIMESTAMPTZ | Session start time |
| `closed_at` | TIMESTAMPTZ | Session end time (null if open) |
| `error_message` | TEXT | Error detail if status=error |

### `np_pty_audit_log`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `source_account_id` | TEXT | Tenant isolation |
| `session_id` | UUID | FK → np_pty_sessions |
| `event_type` | VARCHAR | `session_open` / `session_close` / `resize` / `error` |
| `detail` | JSONB | Event-specific metadata |
| `created_at` | TIMESTAMPTZ | Event timestamp |

## Security

- **No shell injection** — `PTY_CLAWDE_BINARY_PATH` is set at server startup only; no
  per-request override is possible
- **License gate** — every `/sessions` request and WebSocket upgrade requires
  `X-Nself-License` header matching `NSELF_LICENSE_KEY`
- **Tenant isolation** — all rows scoped by `source_account_id`; Hasura RLS enforced
- **Audit trail** — every session event written to `np_pty_audit_log`

## Hasura RLS

```yaml
filter:
  source_account_id:
    _eq: X-Hasura-Source-Account-Id
```

## Docker

```bash
docker run -d \
  --name plugin-pty \
  -p 9100:9100 \
  -e DATABASE_URL="postgres://user:pass@host/db" \
  -e NSELF_LICENSE_KEY="<key>" \
  nself/plugin-pty:latest
```

## Changelog

- **1.0.0** — Initial release (ClawDE bundle, port 9100, PTY bridge + audit log)
