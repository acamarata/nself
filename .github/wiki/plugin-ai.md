# ɳSelf AI Plugin (plugin-ai)

Multi-provider AI inference service. Paid plugin — requires ɳClaw or ClawDE bundle license.

> **Bundles:** ɳClaw · ClawDE
> **License tier:** max
> **Port:** 9002

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install ai
nself plugin start ai
```

Verify health after start:

```bash
curl http://localhost:9002/health
```

## What It Does

Runs a unified LLM proxy and key-pool service. Core capabilities:

- **Multi-provider completions:** OpenAI, Anthropic, Gemini, Groq, Mistral, Ollama, and custom endpoints
- **Gemini Free Pool (GFP):** DB-backed rotation across up to 20 Gemini API keys using `FOR UPDATE SKIP LOCKED`; V2 rotation algorithm in `go/internal/pool/`
- **OpenAI-compatible external API:** drop-in endpoint for tools that speak the OpenAI spec
- **Embeddings + RAG pipeline:** semantic search via `np_ai_response_cache` and embedding checkpoints
- **SSE streaming:** token-by-token streaming for all supported providers
- **Task-class routing:** routes requests to the right provider/model based on task class
- **Caller tokens:** per-caller rate limits and budget controls
- **Usage tracking and cost dashboard:** per-user, per-tenant, per-provider cost aggregation
- **OAuth token manager:** per-user OAuth tokens for Google/Anthropic/OpenAI

The `claw`, `mux`, and `voice` plugins all depend on this plugin.

## License Gate

All routes require a valid nSelf license. The plugin validates the license on startup via `ping.nself.org/license/validate`. Requests without a valid bearer token receive `401 Unauthorized`. The license check cannot be bypassed — it is wired directly into the router middleware.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PORT` | `9002` | Plugin listen port |
| `DATABASE_URL` | — | Postgres connection string (required) |
| `AI_DEFAULT_PROVIDER` | `openai` | Default inference provider |
| `AI_OPENAI_API_KEY` | — | OpenAI API key |
| `AI_ANTHROPIC_API_KEY` | — | Anthropic API key |
| `AI_GEMINI_API_KEY` | — | Google Gemini API key |
| `AI_GROQ_API_KEY` | — | Groq API key |
| `AI_MISTRAL_API_KEY` | — | Mistral API key |
| `AI_LOCAL_BASE_URL` | — | Ollama server URL (for local models) |
| `GEMINI_FREE_KEY_1..20` | — | Gemini pool keys (up to 20) |
| `PLUGIN_AI_ENCRYPTION_KEY` | — | AES key for encrypting stored OAuth tokens |
| `PLUGIN_AI_MONTHLY_BUDGET_USD` | — | Monthly cost budget cap (USD) |
| `PLUGIN_AI_EXTERNAL_API` | — | Enable OpenAI-compatible external API |
| `PLUGIN_AI_EXTERNAL_PORT` | — | External API port (if enabled) |
| `AI_EMBEDDINGS_ENABLED` | `false` | Enable embedding endpoint |
| `AI_EMBEDDINGS_MODEL` | — | Embedding model name |
| `AI_RATE_LIMIT_ENABLED` | `true` | Enable per-caller rate limiting |
| `AI_CACHE_ENABLED` | `false` | Enable response cache |
| `AI_CACHE_TTL_SECONDS` | `3600` | Cache TTL |

Full env var list: `.env.example` in the plugin directory.

## Port

| Port | Purpose |
|------|---------|
| 9002 | AI inference REST API (internal; nginx proxied via `/plugins/ai/`) |

## Database Tables

Core tables added to your Postgres database:

| Table | Purpose |
|-------|---------|
| `np_ai_usage` | Per-request usage log |
| `np_ai_response_cache` | Response cache (optional) |
| `np_ai_gemini_accounts` | Gemini OAuth account pool |
| `np_ai_caller_tokens` | Per-caller auth tokens and rate limits |
| `np_ai_oauth_tokens` | OAuth tokens (encrypted at rest) |
| `np_ai_queue_metrics` | Request queue metrics |
| `np_ai_user_oauth_tokens` | Per-user OAuth tokens |
| `np_ai_user_daily_cost` | Daily cost rollup per user/tenant |
| `np_ai_tenant_daily_cost` | Daily cost rollup per tenant/provider |
| `np_ai_budgets` | Per-user/tenant budget limits |
| `np_ai_local_models` | Local (Ollama) model registry |
| `np_ai_pool_events` | Gemini pool rotation event log |
| `np_ai_source_routing` | Source-based model routing rules |

Row-Level Security (RLS) is enabled on all user-scoped tables. Tenant isolation uses `tenant_id UUID` with Hasura row-filter `{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}`.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/plugins/ai/` | AI inference API (port 9002) |

## API Reference

```
GET  /health                    Health check (no auth required)
POST /chat                      Chat completion (OpenAI-compatible)
POST /complete                  Text completion
POST /embed                     Embeddings
POST /embeddings                Embeddings (alias)
GET  /models                    List available models
GET  /providers                 List configured providers
POST /analyze                   Analyze text
POST /classify                  Classify text into task class
POST /consensus                 Multi-model consensus completion
POST /documents                 Ingest document for RAG
POST /embed                     Generate embeddings
GET  /events                    SSE event stream
POST /feedback                  Submit response feedback
GET  /gemini/pool               Gemini pool status
POST /gemini/keys               Add Gemini key to pool
GET  /gemini/suggest-keys       Suggest key rotation
GET  /admin/pool                Admin: pool health overview
POST /admin/budget/reset        Admin: reset budget counters
GET  /accounts/usage            Per-caller usage summary
GET  /cost/breakdown            Cost breakdown by provider/model
```

All routes except `/health` require a bearer token. Admin routes additionally require the `admin` role claim in the JWT.

## SSRF Guard

All outbound HTTP requests (to OpenAI, Anthropic, Gemini, etc.) pass through the SSRF guard middleware. Private IP ranges (RFC 1918, loopback, link-local) are blocked. Custom provider URLs must resolve to a public IP or they are rejected.

## Gemini Pool

The Gemini Free Pool (GFP) rotates across up to 20 keys using `FOR UPDATE SKIP LOCKED` for concurrency-safe distribution. Keys are stored in `np_ai_gemini_accounts`. Pool events are logged to `np_ai_pool_events` for audit and debugging. Pool cap increase is tracked in E1/E6 scope.

To add a key to the pool:

```bash
curl -X POST http://localhost:9002/gemini/keys \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"key": "AIza..."}'
```

## Gemini OAuth

The AI plugin supports OAuth-connected Google accounts as an alternative to raw API keys. If a token expires or a scope change occurs, re-authorization is required. See [[Plugins-AI-OAuth]] for the full re-auth procedure, token rotation details, and troubleshooting.

## Used By

| Plugin | Usage |
|--------|-------|
| `claw` | AI agent reasoning and memory |
| `mux` | Email and message classification |
| `voice` | Speech transcription and synthesis |

## Troubleshooting

**Plugin fails health check:** verify `DATABASE_URL` is set and Postgres is reachable. The plugin runs migrations on startup — check container logs for migration errors.

**Rate limit errors (429):** a caller token has exceeded its per-minute request or token limit. Adjust limits via `np_ai_caller_tokens` or set `AI_RATE_LIMIT_ENABLED=false` for development.

**Gemini pool exhausted:** all pool keys hit their quota. Add more keys via `POST /gemini/keys` or wait for quota reset (typically midnight Pacific time).

**Cost budget exceeded:** the monthly budget cap (`PLUGIN_AI_MONTHLY_BUDGET_USD`) was reached. Reset via `POST /admin/budget/reset` or raise the cap.

---

See also: [[Plugins-AI-OAuth]] · [[Plugin-Overview]] · [[Home]]
