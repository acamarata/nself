# plugin-clawde

ClawDE daemon integration backend. Manages session lifecycle, daemon health tracking, and
event streaming for the ClawDE AI development environment.

**Port:** 3847 | **Bundle:** ClawDE | **License:** Pro (requires_license=true)

---

## Overview

plugin-clawde bridges the ClawDE desktop AI dev environment with the nSelf backend. The
ClawDE daemon registers a session on startup, sends periodic heartbeats, appends events
(tool calls, model responses, errors), and closes the session on shutdown.

The event log is append-only and scoped by `source_account_id` (Multi-App Isolation
Convention). All data is queryable via Hasura GraphQL with row-level security.

---

## Session Lifecycle

```
ClawDE daemon starts
        │
        ▼
POST /sessions         → creates session, returns session_id
        │
        ▼ (loop)
POST /sessions/:id/heartbeat  → keeps session alive
POST /sessions/:id/events     → appends events
        │
        ▼
DELETE /sessions/:id   → closes session
```

Sessions expire automatically after `NSELF_CLAWDE_SESSION_TTL_MINUTES` of inactivity
(default 60 minutes). Expired sessions are marked `status: expired`.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness check |
| POST | `/sessions` | Open a new daemon session |
| GET | `/sessions/:id` | Get session details |
| POST | `/sessions/:id/heartbeat` | Keep session alive |
| POST | `/sessions/:id/events` | Append event to session log |
| GET | `/sessions/:id/events` | Stream session events (SSE) |
| DELETE | `/sessions/:id` | Close session |
| GET | `/sessions` | List active sessions for account |

### POST /sessions

Request:
```json
{ "metadata": { "version": "1.0.0", "os": "darwin" } }
```

Response (201):
```json
{ "session_id": "abc123", "status": "active", "created_at": "2026-06-25T..." }
```

### POST /sessions/:id/events

Request:
```json
{ "event_type": "tool_call", "payload": "{\"tool\":\"bash\",\"cmd\":\"ls\"}" }
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NSELF_DB_URL` | Yes | — | PostgreSQL connection string |
| `NSELF_CLAWDE_SESSION_TTL_MINUTES` | No | `60` | Session expiry timeout |
| `NSELF_CLAWDE_PORT` | No | `3847` | HTTP server port |
| `NSELF_LICENSE_KEY` | Yes | — | nSelf license key |

---

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_clawde_sessions` | Session lifecycle tracking (active/closed/expired) |
| `np_clawde_events` | Append-only event log per session |

Both tables include `source_account_id TEXT NOT NULL DEFAULT 'primary'` with Hasura row
filters scoping all queries to the requesting account.

Migration notes: sessions use a composite PK `(id, source_account_id)` for strict
multi-account isolation.

---

## Quickstart

```bash
nself plugin install plugin-clawde

# Verify
curl http://localhost:3847/health
# {"status":"ok"}

# Open a session (ClawDE daemon does this automatically)
curl -X POST http://localhost:3847/sessions \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Source-Account-Id: my-account" \
  -d '{"metadata":{"version":"1.0.0"}}'

# Send heartbeat
curl -X POST http://localhost:3847/sessions/abc123/heartbeat \
  -H "X-Hasura-Source-Account-Id: my-account"
```

---

## Security Notes

- Session authentication uses `source_account_id` from `X-Hasura-Source-Account-Id` header
- Cross-account session access is blocked at both handler and Hasura row-filter layers
- Daemon cannot register a session without a valid license key (license gate active on all routes)
- Event payloads are stored as plain text strings — do not include credential material
- Sessions expire automatically; no indefinite open sessions

---

## Integration with ClawDE Desktop

The ClawDE desktop app (`clawde/` repo) connects to this plugin at startup via the nSelf
local daemon address (default `http://localhost:3847`). Session ID is stored in the ClawDE
app state for the duration of the IDE session.

See: `clawde/lib/backend/session_client.dart` for the client implementation.
