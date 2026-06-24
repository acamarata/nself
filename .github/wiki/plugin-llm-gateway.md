# plugin-llm-gateway

**Bundle:** ClawDE · **Port:** 8090 · **License:** Pro (requires_license=true)

The LLM gateway provides ClawDE IDE a simplified, cached, quota-aware LLM API surface. It proxies requests to the upstream `nself-ai-gateway` (port 3761) and adds per-session context injection, Redis-backed response caching, per-tenant daily token quota, and SSRF guard.

---

## Architecture

```
ClawDE IDE
    │
    ▼ POST /v1/completions
plugin-llm-gateway :8090
    │
    ├─► Redis  ── cache lookup / store
    │
    ├─► PostgreSQL  ── quota check (atomic), cost log write, session context read
    │
    └─► nself-ai-gateway :3761  ── upstream LLM proxy (SSRF-guarded)
```

All traffic between ClawDE and external LLM providers passes through `nself-ai-gateway`. `plugin-llm-gateway` never calls external APIs directly — the SSRF guard rejects any configured upstream that resolves outside localhost/private ranges.

---

## Installation

```bash
nself plugin install plugin-llm-gateway
```

Requires a valid ClawDE bundle or ɳSelf+ license.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/completions` | Proxy LLM completions with caching, quota, and context injection |
| GET | `/health` | Health check — returns `{"status":"ok","service":"plugin-llm-gateway","port":8090}` |

### POST /v1/completions

**Request body:**

```json
{
  "model": "gpt-4o-mini",
  "messages": [{"role": "user", "content": "Hello"}],
  "session_id": "my-session",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `model` | string | LLM model identifier (default: `gpt-4o-mini`) |
| `messages` | array | OpenAI-format message array |
| `session_id` | string | Optional — loads session context from DB and prepends as system message |
| `tenant_id` | string | Optional UUID — Cloud SaaS tenant; omit for self-host |

**Headers:**

| Header | Description |
|--------|-------------|
| `X-Source-Account-Id` | Self-host multi-app isolation (default: `primary`) |
| `Authorization` | Forwarded to nself-ai-gateway as-is |

**Response headers:**

| Header | Value | Meaning |
|--------|-------|---------|
| `X-Cache` | `HIT` | Served from Redis cache |
| `X-Cache` | `MISS` | Forwarded to nself-ai-gateway |

**Error codes:**

| Code | Meaning |
|------|---------|
| 400 | Invalid JSON body |
| 403 | SSRF block — upstream URL not on allowlist |
| 429 | Daily token quota exceeded |
| 502 | nself-ai-gateway unreachable |

---

## Configuration

Set via environment variables (injected by `nself start`):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8090` | HTTP listen port |
| `DATABASE_URL` | required | PostgreSQL DSN |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection URL |
| `AI_GATEWAY_URL` | `http://localhost:3761` | nself-ai-gateway URL (must be internal) |
| `PLUGIN_INTERNAL_SECRET` | — | Shared secret forwarded to nself-ai-gateway |
| `PLUGIN_LICENSE_VALID` | required | Must be `true` — injected by nself CLI license gate |
| `DAILY_TOKEN_LIMIT` | `0` | Max tokens per tenant per day (0 = unlimited) |
| `CACHE_TTL_MINUTES` | `5` | Redis cache TTL in minutes |

---

## Quota Model

Token quota is enforced atomically per tenant per calendar day (UTC).

- Cloud SaaS: keyed by `tenant_id` (UUID).
- Self-host: keyed by `source_account_id` (default `"primary"`).
- When `DAILY_TOKEN_LIMIT=0`, usage is tracked but never rejected.
- Quota resets at midnight UTC.
- HTTP 429 `{"error":"daily token quota exceeded"}` returned when limit reached.
- Race-condition safe: uses `UPDATE ... WHERE tokens_used + N <= limit RETURNING tokens_used`; a zero-rowcount result triggers 429.

---

## Response Caching

Identical prompts from the same tenant are cached in Redis.

**Cache key:** `llmgw:resp:` + `SHA256(tenant_key + ":" + model + ":" + messages_json)`

- The tenant key is always included in the hash so tenant A cannot read tenant B's cached responses (CR-C: no cross-tenant poisoning).
- TTL is configurable via `CACHE_TTL_MINUTES` (default 5 minutes).
- Cache hit: `X-Cache: HIT` response header, no upstream call.
- Cache miss: `X-Cache: MISS`, forwarded to nself-ai-gateway, response stored on 200 OK.

---

## Context Injection

When a `session_id` is present in the request, the gateway loads the matching row from `np_llm_gw_sessions` and prepends its `context` field as a system message at the start of the `messages` array.

```
Original messages: [user: "Hello"]
After injection:   [system: "<context>", user: "Hello"]
```

Sessions are managed via the `np_llm_gw_sessions` table through Hasura GraphQL.

---

## SSRF Guard

The `AI_GATEWAY_URL` is validated on every request against an internal-only allowlist:

- Allowed: `localhost`, `127.x.x.x`, `::1`, RFC-1918 private IP ranges (10/8, 172.16/12, 192.168/16).
- Blocked: any external hostname or public IP — returns HTTP 403.

This prevents an attacker from manipulating the gateway URL to exfiltrate data to external LLM APIs.

---

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_llm_gw.np_llm_gw_sessions` | Per-session context storage |
| `np_llm_gw.np_llm_gw_cost_log` | Per-request token usage audit log |
| `np_llm_gw.np_llm_gw_quota_usage` | Per-tenant daily token quota counters |

All tables follow the nSelf Multi-Tenant Convention Wall:
- `source_account_id TEXT NOT NULL DEFAULT 'primary'` for Convention A (self-host isolation).
- `tenant_id UUID` for Convention B (Cloud SaaS multi-tenancy).
- Hasura row filters enforce isolation at the GraphQL layer.

---

## Docker

```bash
docker pull nself/plugin-llm-gateway:latest

docker run -d \
  -p 8090:8090 \
  -e DATABASE_URL="postgres://..." \
  -e REDIS_URL="redis://localhost:6379" \
  -e AI_GATEWAY_URL="http://localhost:3761" \
  -e PLUGIN_LICENSE_VALID="true" \
  nself/plugin-llm-gateway:latest
```

HEALTHCHECK polls `GET /health` every 30s.

---

## Migrations

Migrations are embedded and run automatically on startup.

| File | Description |
|------|-------------|
| `0001_np_llm_gw_sessions.up.sql` | Sessions table |
| `0002_np_llm_gw_cost_log.up.sql` | Cost log table |
| `0003_np_llm_gw_quota_usage.up.sql` | Quota usage table |
| `0003_np_llm_gw_quota_usage.down.sql` | Rollback quota usage table |

---

## Related

- [nself-ai-gateway](plugin-nself-ai-gateway.md) — upstream LLM key pool
- [plugin-clawde](plugin-clawde.md) — ClawDE IDE plugin (PTY relay)
- [ClawDE bundle](https://nself.org/products/clawde) — bundle landing page
