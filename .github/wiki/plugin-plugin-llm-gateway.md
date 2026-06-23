# plugin-llm-gateway

**ClawDE LLM Gateway** — paid plugin, ClawDE bundle, port 8090.

Proxies LLM requests from [ClawDE](https://clawde.io) to `nself-ai-gateway`
(port 3761) with ClawDE session-context injection, per-session cost tracking,
and tenant-scoped data isolation.

## Status

| Field | Value |
|---|---|
| Port | 8090 |
| Bundle | ClawDE |
| Tier | Pro |
| Requires license | Yes |
| Language | Go |
| Upstream dependency | nself-ai-gateway (port 3761) |
| DB tables | `np_llm_gw_sessions`, `np_llm_gw_cost_log` |
| Min nSelf version | 1.1.1 |

## Quick Start

```bash
# Install (requires ClawDE or ɳSelf+ license)
nself license set <your-key>
nself plugin install plugin-llm-gateway
```

Required environment variables (set via `nself env set` or `.env`):

```bash
NSELF_AI_GATEWAY_URL=http://localhost:3761   # upstream nself-ai-gateway
DATABASE_URL=postgres://...                   # Postgres connection
NSELF_PLUGIN_LICENSE_KEY=<your-key>          # ClawDE bundle license
```

## What It Does

### Session Context Injection

ClawDE sends LLM requests with optional session extensions:

```json
{
  "model": "claude-sonnet-4-5",
  "messages": [{"role": "user", "content": "Refactor this"}],
  "x_nself_session": "session-uuid",
  "x_active_file": "internal/handler.go"
}
```

The gateway:

1. Loads the session's `system_prompt_prefix` from `np_llm_gw_sessions`.
2. Prepends a `[ClawDE context]` system message with the prefix + active file.
3. Strips `x_nself_session` and `x_active_file` (not forwarded upstream).
4. Forwards the enriched request to `nself-ai-gateway`.
5. Returns the upstream response.
6. Records token counts + latency to `np_llm_gw_cost_log`.

### SSRF Guard

The upstream gateway URL is read from `NSELF_AI_GATEWAY_URL` at startup.
Request payloads **cannot** override this URL. Any attempt to redirect the
gateway via a request field is silently ignored.

### Cost Tracking

Every LLM call logs to `np_llm_gw_cost_log`:
prompt tokens, completion tokens, model, session ID, tenant ID, latency.

Query via Hasura GraphQL:

```graphql
query SessionCost($sessionId: String!) {
  llm_gw_cost_logs(where: {session_id: {_eq: $sessionId}}) {
    model
    prompt_tokens
    completion_tokens
    latency_ms
    created_at
  }
}
```

## Session Management API

### Create / Update Session

```bash
curl -X POST http://localhost:8090/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-session",
    "system_prompt_prefix": "You are a Go expert working in this codebase.",
    "active_file": "internal/gateway/router.go"
  }'
```

### Get Session

```bash
curl http://localhost:8090/v1/sessions/my-session
```

### Chat Completions

```bash
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Tenant-Id: <tenant-uuid>" \
  -d '{
    "model": "claude-sonnet-4-5",
    "messages": [{"role": "user", "content": "Explain this function"}],
    "x_nself_session": "my-session",
    "x_active_file": "main.go"
  }'
```

## Hasura Row-Level Security

Two tables with tenant isolation:

### np_llm_gw_sessions

| Role | Access | Filter |
|---|---|---|
| `tenant_user` | Full CRUD | `tenant_id = X-Hasura-Tenant-Id` |
| `user` (self-host) | Full CRUD | `tenant_id IS NULL` |
| Admin | Full CRUD | No filter |

### np_llm_gw_cost_log

| Role | Access | Filter |
|---|---|---|
| `tenant_user` | Read-only | `tenant_id = X-Hasura-Tenant-Id` |
| `user` (self-host) | Read-only | `tenant_id IS NULL` |
| Admin | Read-only | No filter |

Writes to `np_llm_gw_cost_log` are Go-service-only (no GraphQL insert permissions).

## Database Migrations

Migrations are in `go/migrations/`:

| File | Purpose |
|---|---|
| `0001_np_llm_gw_sessions.up.sql` | Create session context table |
| `0001_np_llm_gw_sessions.down.sql` | Drop session context table |
| `0002_np_llm_gw_cost_log.up.sql` | Create cost tracking table |
| `0002_np_llm_gw_cost_log.down.sql` | Drop cost tracking table |

Run with `nself db migrate` or directly:

```bash
psql "$DATABASE_URL" -f go/migrations/0001_np_llm_gw_sessions.up.sql
psql "$DATABASE_URL" -f go/migrations/0002_np_llm_gw_cost_log.up.sql
```

## Health Check

```bash
curl http://localhost:8090/health
# {"status":"ok","service":"plugin-llm-gateway"}
```

The health endpoint pings the database — `503` means DB is unreachable.

## Docker

```bash
docker pull nself/plugin-llm-gateway:latest

docker run -p 8090:8090 \
  -e DATABASE_URL="$DATABASE_URL" \
  -e NSELF_AI_GATEWAY_URL="http://host.docker.internal:3761" \
  -e NSELF_PLUGIN_LICENSE_KEY="$NSELF_PLUGIN_LICENSE_KEY" \
  nself/plugin-llm-gateway:latest
```

## Architecture

```
ClawDE client
    |
    | POST /v1/chat/completions
    |   + x_nself_session, x_active_file
    v
plugin-llm-gateway :8090
    |-- load session context (np_llm_gw_sessions)
    |-- inject [ClawDE context] system message
    |-- strip extension fields (SSRF guard)
    |-- POST /v1/chat/completions (clean)
    v
nself-ai-gateway :3761
    |-- provider routing
    v
LLM Provider (Anthropic / OpenAI / Gemini / Ollama)
    |
    v
nself-ai-gateway (response + usage)
    |
    v
plugin-llm-gateway
    |-- write cost log (goroutine, non-blocking)
    v
ClawDE client (response)
```

## Configuration Reference

| Variable | Required | Default | Description |
|---|---|---|---|
| `NSELF_AI_GATEWAY_URL` | Yes | — | nself-ai-gateway base URL (SSRF guard: config-only) |
| `DATABASE_URL` | Yes | — | Postgres DSN |
| `NSELF_PLUGIN_LICENSE_KEY` | Yes | — | ClawDE bundle license key |
| `LLM_GATEWAY_PORT` | No | `8090` | Override listen port |
| `NSELF_LOG_LEVEL` | No | `info` | `debug\|info\|warn\|error` |
| `NSELF_LICENSE_SKIP_VERIFY` | No | — | Set `1` for local dev (key still required) |

## Troubleshooting

### 402 Payment Required

License key missing. Run:

```bash
nself license set <your-key>
```

### 502 Bad Gateway

`nself-ai-gateway` is not running or `NSELF_AI_GATEWAY_URL` is wrong.
Check:

```bash
curl http://localhost:3761/health
```

### 503 Service Unavailable

Database unreachable. Check `DATABASE_URL` and Postgres status.

### Session context not injected

Verify the session exists:

```bash
curl http://localhost:8090/v1/sessions/<session-id>
```

If 404, create it first via `POST /v1/sessions`.

## Related Plugins and Resources

- [nself-ai-gateway](plugin-nself-ai-gateway.md) — upstream LLM provider router (port 3761)
- [Plugin Overview](Plugin-Overview.md) — all plugins
- [Plugin Licensing](Plugin-Licensing.md) — license setup
- SPORT `F04-PLUGIN-INVENTORY-PRO` — plugin registry
- SPORT `F10-PORT-REGISTRY` — port assignments

## Bundle and Pricing

Part of the **ClawDE bundle** (`$0.99/mo · $9.99/yr`).
Also included in **ɳSelf+** (`$3.99/mo · $39.99/yr`).

```bash
# Purchase at
https://nself.org/products/clawde
```
