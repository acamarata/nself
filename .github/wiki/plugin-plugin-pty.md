# PTY Plugin

> PTY bridge management for Claude Max sessions. Bridges pseudo-terminal sessions between the ClawDE daemon and the nSelf backend via WebSocket. **Pro plugin — ClawDE bundle.**

> **Requires:** ClawDE bundle or ɳSelf+. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install plugin-pty
```

## What It Does

plugin-pty is the PTY (pseudo-terminal) bridge service for the ClawDE AI development environment. It connects the ClawDE daemon to Claude Max session processes via WebSocket, forwarding stdin/stdout/stderr between the client and the target process.

Key capabilities:

- WebSocket PTY bridge: bidirectional stdin/stdout/stderr streaming between ClawDE and Claude Max
- Session lifecycle management: create, list, inspect, and close PTY sessions
- Tenant isolation: every session is scoped to the authenticated tenant (Hasura JWT claim)
- Audit log: append-only record of session start, close, and security events
- Session cap: configurable maximum concurrent sessions per tenant
- Hardcoded exec target: the PTY process binary is operator-configured only — clients cannot influence which binary runs

## Security Model

The exec target (what binary the PTY runs) is configured by the nSelf operator via the `PTY_EXEC_PATH` environment variable set at install time. WebSocket clients cannot specify, override, or influence the exec path. Input from WebSocket clients is forwarded verbatim as stdin bytes — no shell interpretation occurs.

Shell injection is blocked at the architecture level: there is no shell involved. The configured binary is exec'd directly with no shell wrapper.

Tenant isolation is enforced at the HTTP handler layer (X-Hasura-Tenant-Id header, validated by Hasura JWT), the database query layer (all queries filter by tenant_id), and the Hasura row filter (row-level security on all np_pty_* tables).

## Configuration

| Env Var | Default | Required | Description |
|---------|---------|----------|-------------|
| `DATABASE_URL` | — | Yes | Postgres connection string |
| `PTY_EXEC_PATH` | — | Yes | Absolute path to Claude Max binary (operator-set) |
| `PTY_PORT` | `9100` | No | Port to listen on |
| `PTY_HOST` | `0.0.0.0` | No | Bind host |
| `PTY_SESSION_TTL_SECONDS` | `3600` | No | Max session duration before auto-close |
| `PTY_MAX_SESSIONS_PER_TENANT` | `5` | No | Concurrent session cap per tenant |
| `PTY_READ_BUFFER_SIZE` | `4096` | No | I/O read buffer size in bytes |
| `PTY_WRITE_BUFFER_SIZE` | `4096` | No | I/O write buffer size in bytes |
| `LOG_LEVEL` | `info` | No | Log verbosity |

`PTY_EXEC_PATH` must be an absolute path to an existing binary. plugin-pty verifies the file exists at startup and exits fatally if it does not. This is intentional — fail fast rather than at session time.

## Ports

| Port | Purpose |
|------|---------|
| 9100 | plugin-pty HTTP/WebSocket service |

## Database Tables

2 tables added to your Postgres database:

- `np_pty_sessions` — active and historical PTY bridge sessions; tenant-scoped
- `np_pty_audit_log` — append-only security audit log; UPDATE and DELETE are blocked by a DB trigger

### np_pty_sessions

| Column | Type | Notes |
|--------|------|-------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Hasura row-filter scope |
| `session_token` | TEXT | Unique token for this session |
| `status` | TEXT | `active`, `closed`, or `error` |
| `created_at` | TIMESTAMPTZ | Session start time |
| `closed_at` | TIMESTAMPTZ | Session end time (nullable) |
| `error_msg` | TEXT | Error description if status=error (nullable) |
| `metadata` | JSONB | Arbitrary session metadata |

### np_pty_audit_log

| Column | Type | Notes |
|--------|------|-------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Hasura row-filter scope |
| `session_id` | UUID | FK to np_pty_sessions (nullable) |
| `event_type` | TEXT | One of: session_created, session_closed, session_error, io_error, auth_denied, injection_blocked |
| `details` | JSONB | Event details |
| `created_at` | TIMESTAMPTZ | Event time |

The audit log is immutable: a DB trigger raises an error on any UPDATE or DELETE attempt.

## Hasura Row Filters

Hasura row-level security is configured on all np_pty_* tables. The `tenant` role has full CRUD on np_pty_sessions and insert+select on np_pty_audit_log, both filtered by:

```yaml
filter:
  tenant_id:
    _eq: X-Hasura-Tenant-Id
```

No update or delete permissions exist on np_pty_audit_log at the Hasura layer (audit immutability is doubly enforced: Hasura permissions + DB trigger).

## API

### Health

```
GET /health
```

Returns `{"status": "ok"}` if the service and database are healthy. Returns 503 on degraded state. No authentication required. Used by Docker HEALTHCHECK every 30 seconds.

### Sessions

```
POST   /sessions          — Create a new PTY session
GET    /sessions          — List sessions for the authenticated tenant
GET    /sessions/{id}     — Get one session by ID
DELETE /sessions/{id}     — Close and delete a session
GET    /sessions/{id}/ws  — WebSocket PTY bridge for a session
```

All session endpoints require a valid bearer JWT and the `X-Hasura-Tenant-Id` header.

### WebSocket Bridge

```
GET /sessions/{id}/ws
Upgrade: websocket
Authorization: Bearer <token>
X-Hasura-Tenant-Id: <tenant-uuid>
```

After upgrade:
- Binary or text frames from the client are forwarded to the PTY process as stdin.
- Output from the PTY process (stdout and stderr) is forwarded to the client as binary frames.
- When the client disconnects or the session TTL expires, the PTY process is killed and the session is marked closed.
- Session audit events are written on connect and disconnect.

### Audit Log

```
GET /audit?limit=100      — List recent audit events for the authenticated tenant
```

Returns the most recent audit events in descending timestamp order. Maximum 500 per request.

## Session Lifecycle

1. Client calls `POST /sessions` — session created with status `active`, audit event `session_created` written.
2. Client connects `GET /sessions/{id}/ws` — WebSocket upgrade, PTY process spawned.
3. Bidirectional I/O flows until client disconnects, process exits, or TTL expires.
4. On close: session marked `closed`, audit event `session_closed` written.
5. On error: session marked `error`, audit event `session_error` written with error details.
6. Client can call `DELETE /sessions/{id}` to explicitly close and clean up.

## Migrations

| File | Description |
|------|-------------|
| `migrations/001_init.sql` | Creates np_pty_sessions + np_pty_audit_log + immutability trigger |
| `migrations/001_init.down.sql` | Drops all np_pty_* tables, indexes, trigger, function |

Migrations are applied automatically by `nself plugin install plugin-pty` via the nSelf migration runner.

## Bundle

plugin-pty is part of the **ClawDE** bundle. It provides the PTY bridge layer that the ClawDE daemon uses to connect Claude Max sessions to the nSelf backend.

Install the full ClawDE bundle:

```bash
nself plugin install plugin-clawde plugin-pty
```

## Troubleshooting

**`PTY_EXEC_PATH not accessible`** — The path set for `PTY_EXEC_PATH` does not exist or is not readable. Verify the Claude Max binary is installed at the configured path and the nself user has execute permission.

**`max concurrent sessions reached`** — The tenant has hit `PTY_MAX_SESSIONS_PER_TENANT`. Close idle sessions with `DELETE /sessions/{id}` or increase the limit via the env var.

**WebSocket upgrade fails** — Ensure the request includes a valid `Authorization: Bearer` header and `X-Hasura-Tenant-Id`. The session must exist and have status `active`.

**Session stuck in `active` after process crash** — Call `DELETE /sessions/{id}` to force-close. The session will be marked `closed` in the audit log.

---

See also: [[plugin-plugin-clawde]] | [[bundle-clawde]] | [[Home]]
