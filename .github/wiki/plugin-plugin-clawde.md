# plugin-clawde

ClawDE daemon integration backend for nSelf. Manages session lifecycle, tracks daemon
health, and streams events for the ClawDE AI development environment.

**Port:** 3847 | **License:** Pro | **Bundle:** ClawDE

---

## What This Plugin Does

ClawDE runs as a desktop/mobile app that communicates with the nSelf backend. This plugin
provides the backend half: it accepts sessions from running ClawDE instances, records
heartbeats to detect when a session is still alive, and stores an append-only event log.

The plugin does NOT include the PTY bridge (that is plugin-pty) or any ClawDE UI logic.
It is the persistence and coordination layer only.

---

## Session Lifecycle

```
create → active → (heartbeats keep alive) → closed or expired
```

| State | Description |
|-------|-------------|
| `active` | Session is running; heartbeats expected every 30-60s |
| `closed` | Session was closed cleanly via DELETE |
| `expired` | Session missed heartbeats for TTL duration (default 60 min) |

Sessions are scoped by `source_account_id`. Different nSelf accounts on the same
deployment never see each other's sessions (Multi-App Isolation Convention).

---

## API Reference

### GET /health

Check plugin health.

```
GET /health
→ {"status": "ok"}         200 — DB reachable
→ {"status": "unhealthy"}  503 — DB unavailable
```

---

### POST /sessions

Create a new active session.

**Request:**

```json
{
  "id": "sess-abc123",
  "source_account_id": "primary"
}
```

**Response (201):**

```json
{
  "id": "sess-abc123",
  "source_account_id": "primary",
  "status": "active",
  "created_at": "2026-01-15T10:00:00Z",
  "last_heartbeat": "2026-01-15T10:00:00Z"
}
```

If a session with the same ID already exists, it is reactivated and its `last_heartbeat`
is reset.

---

### POST /sessions/:id/heartbeat

Keep a session alive. Call every 30-60 seconds from the ClawDE client.

```
POST /sessions/sess-abc123/heartbeat?source_account_id=primary
→ {"status": "ok"}
```

Returns 404 if the session is not found or already closed.

---

### POST /sessions/:id/events

Append an event to the session log. Events are append-only.

**Request:**

```json
{
  "event_type": "tool_call",
  "payload": "{\"tool\":\"bash\",\"cmd\":\"ls -la\"}",
  "source_account_id": "primary"
}
```

**Response (201):**

```json
{"status": "recorded"}
```

Common event types:

| Event type | Description |
|-----------|-------------|
| `session_start` | Session initialized |
| `tool_call` | A tool was invoked |
| `tool_result` | Tool returned result |
| `model_response` | AI model generated response |
| `error` | Error occurred in session |
| `session_end` | Session closing |

---

### DELETE /sessions/:id

Close a session cleanly.

```
DELETE /sessions/sess-abc123?source_account_id=primary
→ {"status": "closed"}
```

Returns 404 if the session is not found or already closed.

---

## Quick Start

```bash
nself license set <your-key>
nself plugin install plugin-clawde
```

Set required environment variables:

```bash
nself plugin env set plugin-clawde NSELF_DB_URL=postgres://...
nself plugin env set plugin-clawde NSELF_LICENSE_KEY=<your-key>
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NSELF_DB_URL` | Yes | — | PostgreSQL connection string |
| `NSELF_LICENSE_KEY` | Yes | — | nSelf Pro license key |
| `NSELF_CLAWDE_PORT` | No | `3847` | HTTP server port |
| `NSELF_CLAWDE_DAEMON_ADDR` | No | `localhost:3848` | ClawDE PTY bridge address |
| `NSELF_CLAWDE_SESSION_TTL_MINUTES` | No | `60` | Session expiry with no heartbeat |

---

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_clawde_sessions` | Session lifecycle state (active/closed/expired) |
| `np_clawde_events` | Append-only event log |

Both tables use composite primary key on `(id, source_account_id)` or
`(session_id, source_account_id)` for tenant isolation.

Hasura row filters enforce `source_account_id = X-Hasura-Source-Account-Id` on the
`nself_user` role for both tables.

---

## Tenant Isolation

All session operations require `source_account_id`. If omitted, it defaults to `"primary"`.
This ensures that accounts sharing a nSelf deployment cannot read each other's sessions or
events — enforced at both the application layer (query WHERE clauses) and the Hasura layer
(row-level permission filters).

---

## License

Requires an active nSelf Pro license. Purchase at [nself.org/pro](https://nself.org/pro).

---

## Related

- [[plugin-nself-ai-gateway]] — AI key pool used by ClawDE sessions
- [[plugin-nself-ai-mcp]] — MCP tools available during ClawDE sessions
- [[Architecture]]
- [[Home]]
