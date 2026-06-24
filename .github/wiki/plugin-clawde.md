# plugin-clawde

ClawDE daemon integration plugin. Bridges the ClawDE desktop AI dev environment with
the nSelf backend for session management, tool registration, and daemon lifecycle.

**Bundle:** ClawDE · **Port:** 3847 · **License required:** Yes (ClawDE bundle)

---

## Install

```bash
nself plugin install plugin-clawde
```

Requires an active ClawDE bundle license. Activate with:

```bash
nself license activate <key>
```

Verify installation:

```bash
nself plugin list
# plugin-clawde   v1.0.0   active   port 3847
```

---

## What It Does

plugin-clawde provides the server-side session layer for ClawDE daemon processes:

- **Session registration** — daemon calls `POST /sessions` on startup to obtain a session ID.
- **Tool catalog** — daemon publishes its available tools (with JSON schemas) via `POST /sessions/{id}/tools`.
- **Heartbeat** — daemon pings `POST /sessions/{id}/heartbeat` every ~30 seconds to stay alive.
- **Expiry** — sessions silent for more than 60 seconds are automatically marked `expired`.
- **Cross-surface visibility** — other nSelf surfaces can query active sessions and tool catalogs.

---

## Daemon Protocol

### Step 1 — Register Session

```http
POST http://localhost:3847/sessions
X-Hasura-Source-Account-Id: primary
Content-Type: application/json

{
  "daemon_id": "clawde-desktop-mac-arm64",
  "metadata": {"version": "1.2.0", "os": "darwin"}
}
```

**Response 201:**
```json
{
  "id": "a1b2c3d4-...",
  "source_account_id": "primary",
  "daemon_id": "clawde-desktop-mac-arm64",
  "status": "active",
  "last_heartbeat": "2026-06-23T10:00:00Z",
  "metadata": {"version": "1.2.0", "os": "darwin"},
  "created_at": "2026-06-23T10:00:00Z",
  "updated_at": "2026-06-23T10:00:00Z"
}
```

### Step 2 — Register Tools

```http
POST http://localhost:3847/sessions/<session-id>/tools
Content-Type: application/json

{
  "tool_name": "read_file",
  "schema": {
    "type": "object",
    "properties": {
      "path": {"type": "string", "description": "Absolute path to file"}
    },
    "required": ["path"]
  }
}
```

**Response 201:** Tool registration record with `id`, `session_id`, `tool_name`, `schema`.

### Step 3 — Send Heartbeats

```http
POST http://localhost:3847/sessions/<session-id>/heartbeat
X-Hasura-Source-Account-Id: primary
```

**Response 200:**
```json
{"session_id": "...", "last_heartbeat": "2026-06-23T10:00:30Z", "status": "ok"}
```

Send every 30 seconds. Session expires after 60 seconds without a heartbeat.

---

## Session Lifecycle

```
startup  →  POST /sessions           →  status: active
                ↓
loop     →  POST /sessions/{id}/heartbeat  (every ~30s)
                ↓
silence > 60s  →  status: expired    (background sweep)
                ↓
shutdown →  DELETE /sessions/{id}    →  removed
```

---

## API Reference

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | None | Health check |
| POST | `/sessions` | X-Hasura-Source-Account-Id | Register daemon session |
| GET | `/sessions` | X-Hasura-Source-Account-Id | List active sessions |
| GET | `/sessions/{id}` | X-Hasura-Source-Account-Id | Get session details |
| DELETE | `/sessions/{id}` | X-Hasura-Source-Account-Id | Remove session |
| POST | `/sessions/{id}/tools` | X-Hasura-Source-Account-Id | Register/update a tool |
| GET | `/sessions/{id}/tools` | X-Hasura-Source-Account-Id | List tools for session |
| DELETE | `/sessions/{id}/tools/{name}` | X-Hasura-Source-Account-Id | Remove a tool |
| POST | `/sessions/{id}/heartbeat` | X-Hasura-Source-Account-Id | Keep session alive |

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | Postgres DSN |
| `NSELF_LICENSE_KEY` | Yes | — | ClawDE bundle license key |
| `CLAWDE_PORT` | No | `3847` | Listen port |
| `CLAWDE_ALLOWED_ORIGINS` | No | `` | CORS allowed origins (CSV) |
| `CLAWDE_SESSION_TTL_SECONDS` | No | `60` | Seconds before silent session expires |

---

## Database Tables

| Table | Purpose |
|---|---|
| `np_clawde_sessions` | Registered daemon sessions |
| `np_clawde_tool_registrations` | Tools published by each session |
| `np_clawde_heartbeats` | Heartbeat audit log |

All tables use `source_account_id` for multi-app isolation (not `tenant_id` — see Convention Wall in PPI).

---

## Docker

```bash
docker pull nself/plugin-clawde:latest

docker run \
  -e DATABASE_URL=postgresql://... \
  -e NSELF_LICENSE_KEY=your-key \
  -p 3847:3847 \
  nself/plugin-clawde:latest
```

Health check endpoint: `GET http://localhost:3847/health`

---

## Security

- **License gate:** service refuses to start without `NSELF_LICENSE_KEY`. Unauthenticated
  daemons cannot register sessions.
- **Tenant isolation:** every DB query filters by `source_account_id`. A request without
  the header defaults to `'primary'`. Cross-account data is unreachable by construction.
- **Schema sanitization:** tool schemas are parsed as JSON objects and re-serialized before
  storage. Non-object values (arrays, strings, numbers) are rejected with HTTP 400.
- **Hasura RLS:** all three tables enforce `source_account_id = X-Hasura-Source-Account-Id`
  on the `nself_user` role, providing a second enforcement layer at the GraphQL layer.

---

## Uninstall

```bash
nself plugin remove plugin-clawde
```

This stops the container and removes the plugin config. The database tables
(`np_clawde_*`) are preserved. To drop them, run the down migration:

```sql
\i plugins-pro/paid/plugin-clawde/migrations/001_init.down.sql
```

---

## Troubleshooting

**Session not found after restart:**
Sessions are stored in Postgres. If the database was wiped, re-register the daemon.

**Session expires immediately:**
Check that the daemon is sending heartbeats every 30 seconds and that `CLAWDE_SESSION_TTL_SECONDS`
is not set too low.

**License key rejected:**
Run `nself license status` to verify the ClawDE bundle is active on your account.

**Tool schema rejected (HTTP 400):**
Ensure the `schema` field is a JSON object (`{...}`), not an array or primitive value.

---

## Related

- `plugin-pty` — PTY bridge management (T07)
- `plugin-llm-gateway` — ClawDE LLM gateway (T08)
- ClawDE desktop app: `clawde/` repo (out of scope for this plugin)
- SPORT: `F04-PLUGIN-INVENTORY-PRO.md`, `F10-PORT-REGISTRY.md`
