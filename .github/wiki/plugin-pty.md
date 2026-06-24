# PTY Bridge Plugin

> PTY (pseudo-terminal) bridge for ClawDE — spawn, relay, and manage terminal sessions for Claude Max within your development environment. **Pro plugin — requires license.**

> **Requires:** ClawDE bundle or ɳSelf+. `nself license set nself_pro_...`

## Bundle

This plugin is part of the **ClawDE** bundle.

| Bundle | Monthly | Annual | Includes plugin-pty? |
|--------|---------|--------|---------------------|
| ClawDE | $0.99/mo | $9.99/yr | Yes |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

All other bundles (ɳClaw, ɳChat, ɳTV, ɳFamily): No.

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install plugin-pty
nself start
```

## What It Does

`plugin-pty` provides the PTY (pseudo-terminal) bridge that ClawDE uses to spawn interactive shell sessions for Claude Max. When ClawDE opens a terminal, it calls this plugin to create a PTY process server-side, then streams I/O bidirectionally over WebSocket.

Each PTY session is isolated per tenant. A configurable per-tenant limit (default: 5 concurrent sessions) prevents resource exhaustion. Sessions are tracked in Postgres and visible in the ɳSelf admin panel.

The plugin exposes a minimal REST + WebSocket API:

- `POST /sessions` spawns a new PTY and returns a WebSocket URL.
- `GET /sessions/:id/io` provides the bidirectional I/O relay (WebSocket).
- `DELETE /sessions/:id` terminates the PTY and marks the session closed.

## Security

PTY sessions are strictly isolated per tenant. Every request, including WebSocket upgrade, validates `X-Tenant-Id`. A mismatch returns `403` before any I/O occurs.

The per-tenant session limit is enforced atomically under a mutex. No timing window allows bursting above the cap.

Shell spawning uses a fixed path (`/bin/sh`). No user-supplied shell path is accepted.

All endpoints except `/health` require `Authorization: Bearer <NSELF_INTERNAL_SECRET>`. The comparison uses constant-time equality to prevent timing attacks.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL DSN. Omit to run in dev mode (in-memory only). |
| `NSELF_INTERNAL_SECRET` | — | Bearer token gate (required in production). |
| `PORT` | `9100` | Listen port. |
| `HOST` | `0.0.0.0` | Bind address. |
| `PTY_MAX_PER_TENANT` | `5` | Max concurrent PTY sessions per tenant. |
| `PTY_IDLE_TIMEOUT_SECONDS` | `300` | Reserved: idle auto-close (seconds). |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error`. |

## Ports

| Port | Purpose |
|------|---------|
| 9100 | PTY bridge REST + WebSocket API |

## REST API

All endpoints require `Authorization: Bearer <NSELF_INTERNAL_SECRET>` except `/health`.

### GET /health

Health check for Docker HEALTHCHECK and load balancer probes. No auth required.

```json
{"status": "ok", "plugin": "plugin-pty", "port": 9100}
```

### POST /sessions

Spawn a new PTY session.

Headers:

- `X-Tenant-Id: <uuid>` (required)
- `X-Source-Account-Id: <string>` (optional, defaults to `primary`)

Response `201`:

```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "ws_url": "/sessions/550e8400-e29b-41d4-a716-446655440000/io"
}
```

Error codes:

| Code | Reason |
|------|--------|
| `400` | Missing `X-Tenant-Id` |
| `401` | Missing or invalid `Authorization` |
| `429` | Tenant has reached `PTY_MAX_PER_TENANT` |
| `500` | PTY spawn failure (OS error) |

### DELETE /sessions/:id

Terminate a PTY session. Returns `204 No Content`.

| Code | Reason |
|------|--------|
| `403` | Caller's `X-Tenant-Id` does not match the session owner |
| `404` | Session not found |

### GET /sessions/:id/io (WebSocket)

Upgrades to WebSocket for bidirectional I/O relay.

- **Server to client:** raw terminal output bytes from the PTY master fd.
- **Client to server:** raw keyboard input bytes written to the PTY.

The connection closes when the PTY process exits or the client disconnects. Returns HTTP `403` before upgrade if the tenant does not own the session.

## Database Tables

1 table added to your Postgres database:

- `np_pty_sessions` — tracks active and closed PTY sessions (tenant, PID, status, timestamps)

Hasura row-filter enforces tenant isolation: `tenant_id = X-Hasura-Tenant-Id` and `source_account_id = X-Hasura-Source-Account-Id`.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/pty/` | plugin-pty REST API (port 9100) |
| `/pty/sessions/:id/io` | WebSocket upgrade for PTY I/O relay |

## Examples

### Spawn a session and connect via wscat

```bash
# Spawn
curl -X POST http://localhost:9100/sessions \
  -H "Authorization: Bearer $NSELF_INTERNAL_SECRET" \
  -H "X-Tenant-Id: my-tenant-uuid"

# Connect I/O
wscat -c "ws://localhost:9100/sessions/<session_id>/io" \
  -H "Authorization: Bearer $NSELF_INTERNAL_SECRET" \
  -H "X-Tenant-Id: my-tenant-uuid"
```

### Terminate a session

```bash
curl -X DELETE http://localhost:9100/sessions/<session_id> \
  -H "Authorization: Bearer $NSELF_INTERNAL_SECRET" \
  -H "X-Tenant-Id: my-tenant-uuid"
```

### Health check

```bash
curl http://localhost:9100/health
# {"status":"ok","plugin":"plugin-pty","port":9100}
```

### Check the tenant session limit

```bash
# Spawn 6 sessions — the 6th returns 429 when PTY_MAX_PER_TENANT=5
for i in $(seq 1 6); do
  curl -s -o /dev/null -w "%{http_code}\n" \
    -X POST http://localhost:9100/sessions \
    -H "Authorization: Bearer $NSELF_INTERNAL_SECRET" \
    -H "X-Tenant-Id: test-tenant"
done
# 201 201 201 201 201 429
```

## Source Code

Source-available, license-gated: `plugins-pro/paid/plugin-pty/` (private GitHub repo, available to ClawDE bundle subscribers).

## See Also

- [[plugin-clawde]] — ClawDE integration plugin
- [[bundle-clawde]] — full ClawDE bundle overview
- [[plugin-llm-gateway]] — LLM routing for ClawDE sessions

---

← [[Plugins]] | [[Home]] →
