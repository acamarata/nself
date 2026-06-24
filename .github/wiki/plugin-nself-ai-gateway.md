# Plugin: nself-ai-gateway

**Port:** 3761 | **Bundle:** ɳClaw | **Tier:** Pro | **Category:** integrations

The `nself-ai-gateway` plugin provides a shared LLM key pool for the ɳClaw AI stack. It routes requests to multiple LLM providers (Anthropic Claude, OpenAI, Google Gemini, Ollama, vLLM, TEI) with AES-256-GCM key encryption, lane affinity dispatch, per-tenant isolation, and OpenTelemetry observability.

---

## Architecture

```
Request
  │
  ├─ /health              → 200 OK (no auth)
  │
  └─ License Gate         → ping.nself.org validates NSELF_LICENSE_KEY
       │
       ├─ Lane Classifier  → fast | deep | multimodal | embedding | rerank | live | local
       │
       └─ Key Pool (ag_key_pool)
            │
            ├─ LRU + tenant-affinity selection (FOR UPDATE SKIP LOCKED)
            │
            ├─ AES-256-GCM decrypt (in-memory only, never logged)
            │
            ├─ Provider dispatch (Anthropic | OpenAI | Google | Ollama | vLLM | TEI)
            │
            ├─ Audit write (ag_key_audit — key_id UUID only, no plaintext)
            │
            └─ OTLP span (lane_dispatch / key_rotation)
```

---

## Lane Affinity

Lane affinity is the mechanism that routes each request type to appropriate API keys. Keys with `lane_affinity = NULL` serve all lanes.

| Lane | Request type | Example models |
|---|---|---|
| `fast` | Low-latency chat + streaming | claude-3-5-haiku-latest, gpt-4o-mini, gemini-1.5-flash |
| `deep` | Complex reasoning, large context, long-form generation | claude-opus-4-5, gpt-4o, llama-3-70b-instruct |
| `multimodal` | Image understanding, vision tasks | gpt-4-vision-preview, gemini-pro-vision |
| `embedding` | Vector embedding generation for RAG / search | text-embedding-3-small, nomic-embed-text |
| `rerank` | Semantic reranking of retrieval results | bge-reranker-v2-m3, cross-encoder/ms-marco |
| `live` | Real-time streaming with low-latency SLA | claude-3-5-haiku-latest (streaming), gpt-4o |
| `local` | On-premise inference — no cloud API cost | llama3, mistral, phi-3 (via Ollama or vLLM) |

**Classification priority** (first match wins):
1. Explicit `task_type` header/field
2. Model name embedding/rerank hint
3. Provider is `ollama`/`vllm`/`tei` → local
4. Model name large-model hint (opus, 70b, pro, ultra, large) → deep
5. Model name vision hint → multimodal
6. Default → fast

---

## Key Rotation

Keys rotate automatically using LRU (least recently used) selection with exponential backoff on errors.

### Exponential Backoff Formula

```
cooldown_until = now() + min(2^error_count × 60s, 3600s)
```

| Consecutive errors | Cooldown |
|---|---|
| 1 | 120s |
| 2 | 240s |
| 3 | 480s |
| 4 | 960s |
| 5 | 1920s |
| ≥ 6 | 3600s (cap) |

**Auto-recovery:** One successful health-check probe resets `error_count = 0` and clears `cooldown_until` immediately.

**Pool cap:** Maximum 30 keys per provider (configurable via `NSELF_AI_POOL_MAX_KEYS`).

---

## Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/health` | None | Returns `{"status":"ok"}` — used by Docker HEALTHCHECK |
| `POST` | `/v1/chat/completions` | License | OpenAI-compatible chat completions |
| `POST` | `/v1/embeddings` | License | Embedding generation (lane: embedding) |
| `POST` | `/v1/rerank` | License | Semantic reranking (lane: rerank) |
| `GET` | `/v1/models` | License | List available model IDs |
| `POST` | `/v1/keys` | License | Register a new provider API key |
| `DELETE` | `/v1/keys/{keyID}` | License | Disable a key (soft delete) |

Authentication: `Authorization: Bearer <NSELF_LICENSE_KEY>` or `X-Nself-License: <key>`.

---

## Environment Variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `NSELF_VAULT_KEY` | Yes | — | 32-byte hex-encoded AES-256 key for key encryption |
| `NSELF_LICENSE_KEY` | Yes | — | nSelf license key (validated at ping.nself.org) |
| `PLUGIN_INTERNAL_SECRET` | Yes | — | Shared secret for inter-plugin calls |
| `PORT` | No | `3761` | HTTP listen port |
| `NSELF_OTLP_ENDPOINT` | No | `` | OTLP gRPC endpoint (empty = no tracing) |
| `NSELF_AI_POOL_MAX_KEYS` | No | `30` | Max API keys per provider pool |
| `NSELF_AI_HEALTH_INTERVAL_SEC` | No | `60` | Provider health-check interval (seconds) |
| `NSELF_AI_AUDIT_RETENTION_DAYS` | No | `90` | Audit log retention period (days) |
| `NSELF_PING_API_URL` | No | `https://ping.nself.org` | License validation endpoint |

---

## Quickstart

### Install

```bash
nself plugin install nself-ai-gateway
```

### Generate Vault Key

```bash
openssl rand -hex 32
```

### Register Keys

```bash
# Register an Anthropic key for fast + deep lanes
curl -X POST http://localhost:3761/v1/keys \
  -H "Authorization: Bearer $NSELF_LICENSE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "anthropic",
    "key": "sk-ant-api03-...",
    "lane_affinity": ["fast", "deep"]
  }'

# Register an OpenAI key for embedding lane
curl -X POST http://localhost:3761/v1/keys \
  -H "Authorization: Bearer $NSELF_LICENSE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "key": "sk-proj-...",
    "lane_affinity": ["embedding"]
  }'
```

### Send a Request

```bash
curl -X POST http://localhost:3761/v1/chat/completions \
  -H "Authorization: Bearer $NSELF_LICENSE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-haiku-latest",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### Tenant Isolation

For Cloud multi-tenant deployments, pass `X-Hasura-Tenant-Id` to scope key selection:

```bash
curl -X POST http://localhost:3761/v1/chat/completions \
  -H "Authorization: Bearer $NSELF_LICENSE_KEY" \
  -H "X-Hasura-Tenant-Id: <tenant-uuid>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hi"}]}'
```

---

## Security Model

- **Encryption at rest:** API keys stored as `base64(nonce[12] || AES-256-GCM(key))`. The vault key (`NSELF_VAULT_KEY`) is the only decryption secret.
- **Decrypt at dispatch:** Plaintext keys exist only in memory for the duration of the outbound HTTP call. Never logged, never stored, never included in traces.
- **Audit log safety:** `ag_key_audit` records `key_id` UUID only — no key values.
- **OTLP span safety:** Spans include `key.id` UUID, provider name, and lane — never key material.
- **Cross-tenant isolation:** Key selection filters by `tenant_id = $tenantID OR tenant_id IS NULL`. A tenant can never access another tenant's dedicated keys.
- **License fail-open:** If `ping.nself.org` is unreachable, validation fails open (7-day cached TTL). This prevents accidental self-DoS on network blips.

---

## Database Tables

### ag_key_pool

Encrypted provider API keys with rotation state.

```sql
SELECT id, provider, tier, enabled, error_count, cooldown_until, lane_affinity, tenant_id
FROM ag_key_pool
WHERE provider = 'anthropic' AND enabled = true
ORDER BY last_used_at ASC NULLS FIRST;
```

### ag_key_audit

Per-call cost accounting. Append-only; 90-day retention.

```sql
SELECT provider, caller, SUM(tokens_used), SUM(cost_usd)
FROM ag_key_audit
WHERE timestamp > now() - INTERVAL '7 days'
GROUP BY provider, caller
ORDER BY SUM(cost_usd) DESC;
```

---

## Observability

OTLP traces are emitted to `NSELF_OTLP_ENDPOINT` (gRPC). Two span types:

- `lane_dispatch` — emitted on every key selection; attributes: `dispatch.key_id`, `dispatch.provider`, `dispatch.lane`.
- `key_rotation` — emitted on key register/disable; attributes: `key.id`, `key.provider`, `rotation.success`.

No key material appears in any span attribute.

---

## CLI Reference

```bash
nself plugin install nself-ai-gateway    # Install
nself plugin status nself-ai-gateway     # Check health
nself plugin logs nself-ai-gateway       # View logs
nself plugin remove nself-ai-gateway     # Uninstall
```

---

## Changelog

| Version | Change |
|---|---|
| 0.1.0 | Initial release: ag_key_pool + ag_key_audit, AES-256-GCM, 7-lane affinity, OTLP spans, license gate |

---

See also: [nself.org/products/claw](https://nself.org/products/claw) — ɳClaw bundle overview.
