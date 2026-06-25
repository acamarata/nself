# plugin-llm-gateway

> ClawDE-facing LLM gateway: per-tenant token quota, Redis-backed response caching, session context injection, and SSRF guard over nself-ai-gateway (port 3761). **Pro plugin — requires license.**

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install plugin-llm-gateway
```

## Architecture

```
ClawDE client
     |
     v
plugin-llm-gateway :8090
     |  - quota check (DB)
     |  - cache lookup (in-memory / Redis)
     |  - context injection (session system prompt)
     |  - SSRF guard
     v
nself-ai-gateway :3761 (key pool, provider routing)
     |
     v
LLM providers (OpenAI, Anthropic, etc.)
```

The gateway is intentionally thin: it adds operational concerns (quota, cache, context) on top of `nself-ai-gateway` without duplicating provider logic.

## Endpoints

### `POST /v1/completions`

Forward a completion request to nself-ai-gateway with quota enforcement and caching.

**Request:**
```json
{
  "model": "gpt-4o",
  "messages": [{"role": "user", "content": "Hello"}],
  "source_account_id": "primary",
  "session_id": "sess-abc123"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `model` | Yes | LLM model name (passed to nself-ai-gateway) |
| `messages` | Yes | OpenAI-compatible message array |
| `source_account_id` | No | Multi-app isolation key (default: `"primary"`) |
| `session_id` | No | If set, fetches session context for system-message injection |

**Response headers:**
- `X-Cache: HIT` — response served from cache
- `X-Cache: MISS` — response fetched from upstream

**Error codes:**
- `429` — tenant daily token quota exceeded
- `403` — SSRF blocked (gateway URL targets external host)
- `502` — upstream nself-ai-gateway unreachable

### `GET /health`

Returns `{"status":"ok"}` when Postgres is reachable.

## Token Quota

Daily token quota is enforced per `source_account_id`. The default quota is 100,000 tokens/day. Quota increments are atomic (PostgreSQL `ON CONFLICT DO UPDATE` upsert) to prevent race-condition bypass.

| Env Var | Default | Description |
|---------|---------|-------------|
| `DAILY_TOKEN_QUOTA` | `100000` | Max tokens per tenant per day (0 = unlimited) |

When the quota is exceeded, the gateway returns HTTP 429 with:
```json
{"error": "quota_exceeded"}
```

## Response Caching

Responses are cached in-memory with a configurable TTL. The cache key is `SHA256(source_account_id + model + request body)`. Including `source_account_id` in the key prevents cross-tenant cache poisoning.

| Env Var | Default | Description |
|---------|---------|-------------|
| `CACHE_TTL_SECONDS` | `300` | Cache TTL in seconds |

## Context Injection

If `session_id` is provided, the gateway fetches the most recent `context` string for that session from `np_llm_gateway_requests` and prepends it as a system message before forwarding to the upstream gateway.

## SSRF Guard

The upstream gateway URL is validated before every request. Only `localhost`, `127.0.0.1`, `::1`, and RFC-1918 addresses are allowed. External URLs return HTTP 403 immediately.

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_llm_gateway_requests` | Per-request audit log with session context |
| `np_llm_gateway_quota_usage` | Daily token quota per tenant (atomic upsert) |

Both tables use `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation. Hasura row-level filters restrict each tenant to its own rows.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string |
| `NSELF_AI_GATEWAY_URL` | `http://127.0.0.1:3761` | Upstream nself-ai-gateway URL |
| `PORT` | `8090` | HTTP bind port |
| `DAILY_TOKEN_QUOTA` | `100000` | Daily token limit per tenant |
| `CACHE_TTL_SECONDS` | `300` | Response cache TTL |

## Port

| Port | Purpose |
|------|---------|
| 8090 | HTTP API (completions, health) |

## License

Requires `clawde` bundle or ɳSelf+ subscription.

```bash
nself license set nself_pro_xxxxx...
nself plugin install plugin-llm-gateway
nself plugin status plugin-llm-gateway
```

## Docker

```bash
docker pull nself/plugin-llm-gateway:latest
docker run \
  -e DATABASE_URL=postgres://... \
  -e NSELF_AI_GATEWAY_URL=http://nself-ai-gateway:3761 \
  -p 8090:8090 \
  nself/plugin-llm-gateway:latest
```
