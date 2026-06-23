# plugin-llm-gateway

**ClawDE LLM Gateway** — paid plugin, ClawDE bundle, port 8090.

Proxies LLM requests from [ClawDE](https://clawde.io) to `nself-ai-gateway`
(port 3761) with ClawDE session-context injection, per-tenant daily token quota
enforcement, Redis response caching, and tenant-scoped data isolation.

## Status

| Field | Value |
|---|---|
| Port | 8090 |
| Bundle | ClawDE |
| Tier | Pro |
| Requires license | Yes |
| Language | Go |
| Upstream dependency | nself-ai-gateway (port 3761) |
| Redis | Optional (caching disabled when absent) |
| DB tables | `np_llm_gw_sessions`, `np_llm_gw_cost_log`, `np_llm_gw_quota_usage` |
| Min nSelf version | 1.1.1 |

## Architecture

```
ClawDE client
     │
     ▼  POST /v1/chat/completions (or /v1/completions)
plugin-llm-gateway :8090
     │
     ├─ 1. Quota check: np_llm_gw_quota_usage (429 if over daily limit)
     ├─ 2. Cache lookup: Redis SHA-256(tenantID+model+messages)
     ├─ 3. Session context: np_llm_gw_sessions → inject system message
     ├─ 4. SSRF guard: forward ONLY to NSELF_AI_GATEWAY_URL (config-only)
     │
     ▼  POST /v1/chat/completions
nself-ai-gateway :3761 → LLM provider
     │
     ├─ 5. Cache response in Redis (TTL=LLM_CACHE_TTL_SECONDS)
     ├─ 6. Track tokens: np_llm_gw_cost_log
     └─ 7. Return response to ClawDE client
```

## Quick Start

```bash
# Install (requires ClawDE or nSelf+ license)
nself license set <your-key>
nself plugin install plugin-llm-gateway
```

Required environment variables (set via `nself env set`):

```bash
nself env set NSELF_AI_GATEWAY_URL=http://localhost:3761
nself env set DATABASE_URL=postgresql://user:pass@localhost:5432/nself
nself env set NSELF_PLUGIN_LICENSE_KEY=<your-key>
```

Optional variables:

```bash
nself env set REDIS_URL=redis://localhost:6379           # enable response caching
nself env set LLM_QUOTA_DAILY_TOKENS=500000              # daily token limit per tenant (0=unlimited)
nself env set LLM_CACHE_TTL_SECONDS=300                  # cache TTL in seconds (default 300)
nself env set LLM_GATEWAY_PORT=8090                      # override listen port
nself env set NSELF_LOG_LEVEL=info                       # debug|info|warn|error
```

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check (DB ping) |
| `POST` | `/v1/chat/completions` | ClawDE LLM proxy (primary) |
| `POST` | `/v1/completions` | Alias for `/v1/chat/completions` |
| `POST` | `/v1/embeddings` | Pass-through embeddings proxy |
| `GET` | `/v1/sessions/{id}` | Get ClawDE session context |
| `POST` | `/v1/sessions` | Create or update session context |

## ClawDE Session Context

ClawDE requests can carry two optional extension fields:

```json
{
  "model": "claude-sonnet-4-5",
  "messages": [{"role": "user", "content": "Refactor this function"}],
  "x_nself_session": "session-uuid",
  "x_active_file": "src/main.go"
}
```

- `x_nself_session`: Loads the session's `system_prompt_prefix` from `np_llm_gw_sessions` and prepends it as a system message.
- `x_active_file`: Appends the active file path to the system context.

Extension fields are stripped before forwarding to `nself-ai-gateway`.

## Token Quota Enforcement

Set `LLM_QUOTA_DAILY_TOKENS` to enable daily token limits per tenant:

```bash
nself env set LLM_QUOTA_DAILY_TOKENS=100000
```

When a tenant exceeds their daily limit, the gateway returns:

```json
HTTP 429 Too Many Requests
{
  "error": {
    "type": "quota_exceeded",
    "message": "daily token quota exceeded for this tenant"
  }
}
```

Self-hosted deployments (no `X-Hasura-Tenant-Id` header) are never quota-limited.
Quota resets at UTC midnight (per-day bucket in `np_llm_gw_quota_usage`).

## Response Caching

When `REDIS_URL` is set, identical requests (same tenant, model, and messages)
return the cached response on the second call:

```
X-LLM-Cache: HIT   # cached response
X-LLM-Cache: MISS  # forwarded to upstream
```

Cache key: `SHA-256(tenantID + "\x00" + model + "\x00" + messagesJSON)`

The tenant ID is always included in the hash to prevent cross-tenant cache
poisoning (a malicious tenant cannot read another tenant's cached response
by crafting identical model and messages).

## SSRF Guard

The upstream gateway address (`NSELF_AI_GATEWAY_URL`) is read from environment
at startup and never from request payloads. A request body that includes a
`gateway_url` field is ignored. All outbound calls go exclusively to the
configured internal address.

## Database Tables

### np_llm_gw_sessions

Stores per-ClawDE-session context (system prompt prefix + active file).

| Column | Type | Description |
|---|---|---|
| `id` | TEXT PK | ClawDE session ID |
| `tenant_id` | UUID | NULL=self-host |
| `source_account_id` | TEXT | Multi-app isolation |
| `system_prompt_prefix` | TEXT | Injected as system message |
| `active_file` | TEXT | Active editor file hint |
| `created_at` / `updated_at` | TIMESTAMPTZ | Auto-managed |

### np_llm_gw_cost_log

Append-only log of every LLM call with token counts and latency.

| Column | Type | Description |
|---|---|---|
| `id` | BIGSERIAL PK | Auto-incrementing |
| `session_id` | TEXT FK | References np_llm_gw_sessions |
| `tenant_id` | UUID | NULL=self-host |
| `source_account_id` | TEXT | Multi-app isolation |
| `model` | TEXT | Model name |
| `prompt_tokens` | INT | Input token count |
| `completion_tokens` | INT | Output token count |
| `latency_ms` | BIGINT | Wall-clock ms |
| `created_at` | TIMESTAMPTZ | UTC timestamp |

### np_llm_gw_quota_usage

Daily token usage tracking per tenant (one row per tenant per UTC day).

| Column | Type | Description |
|---|---|---|
| `id` | BIGSERIAL PK | Auto-incrementing |
| `tenant_id` | UUID | Cloud tenant (NOT NULL) |
| `quota_date` | DATE | UTC day bucket |
| `source_account_id` | TEXT | Multi-app isolation |
| `tokens_used` | BIGINT | Cumulative daily tokens |
| `created_at` / `updated_at` | TIMESTAMPTZ | Auto-managed |

## Hasura Permissions

All three tables use role-based row filters:

| Role | Access | Filter |
|---|---|---|
| `tenant_user` | Read (cost_log, quota_usage) / CRUD (sessions) | `tenant_id = X-Hasura-Tenant-Id` AND `source_account_id = X-Hasura-Source-Account-Id` |
| `user` | Read sessions + cost_log | `tenant_id IS NULL` AND `source_account_id = X-Hasura-Source-Account-Id` |
| `admin` | Unrestricted | None |

Mutations on `np_llm_gw_cost_log` and `np_llm_gw_quota_usage` are
Go-service-only (no Hasura insert/update/delete permissions for any role).

## Health Check

```bash
curl http://localhost:8090/health
# {"status":"ok","service":"plugin-llm-gateway"}
```

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `402 license_required` | `NSELF_PLUGIN_LICENSE_KEY` not set | `nself license set <key>` |
| `502 upstream_error` | `nself-ai-gateway` unreachable | Verify `NSELF_AI_GATEWAY_URL` + `nself plugin status nself-ai-gateway` |
| `429 quota_exceeded` | Daily token limit reached | Increase `LLM_QUOTA_DAILY_TOKENS` or wait for UTC midnight reset |
| `503 db_unavailable` | Postgres not reachable | Check `DATABASE_URL` and `nself status` |
| Cache always MISS | `REDIS_URL` not set | Set `REDIS_URL` or accept no-cache mode |

## Related Pages

- [[plugin-nself-ai-gateway]] — upstream LLM key pool (required dependency)
- [[plugin-clawde]] — ClawDE plugin suite
- [[plugin-plugin-clawde]] — ClawDE paid plugin
- [[Home]]
