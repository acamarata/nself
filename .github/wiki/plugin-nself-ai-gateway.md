# nself-ai-gateway Plugin

> AI-CP (centralized provider) gateway. OpenAI-compatible API that routes LLM
> requests to multiple providers through an encrypted, per-tenant key pool. **Pro plugin.**

> **Requires:** ɳClaw bundle or ɳSelf+. `nself license set nself_pro_xxxxx...`

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install nself-ai-gateway
nself restart
```

Before the first start, generate a vault key and add it to your `.env.dev`:

```bash
echo "NSELF_VAULT_KEY=$(openssl rand -hex 32)" >> .env.dev
```

---

## What It Does

nself-ai-gateway eliminates per-service LLM key management. All inference
traffic flows through one gateway that:

- Routes requests to the right provider (Anthropic, Google, OpenAI, Ollama, vLLM, TEI)
- Selects the best available key using round-robin + LRU with automatic failover
- Encrypts all provider API keys with AES-256-GCM before database storage
- Isolates dedicated tenant keys from the shared pool
- Tracks cost and token usage per call in `ag_key_audit`
- Enforces SSRF protection (only allow-listed provider hosts are contacted)

The API is OpenAI-compatible: you can point any OpenAI SDK at port 3761 and it works.

---

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `AI_GATEWAY_PORT` | `3761` | HTTP listen port |
| `DATABASE_URL` | — | Postgres URL (REQUIRED) |
| `NSELF_VAULT_KEY` | — | 32-byte hex AES-256 key (REQUIRED) |
| `NSELF_PLUGIN_LICENSE_KEY` | — | Pro license key (REQUIRED) |
| `NSELF_LOG_LEVEL` | `info` | debug / info / warn / error |
| `NSELF_AI_POOL_MAX_KEYS` | `30` | Keys per provider pool |
| `NSELF_AI_HEALTH_INTERVAL_SEC` | `60` | Provider probe interval (s) |
| `NSELF_AI_AUDIT_RETENTION_DAYS` | `90` | Audit log retention window |
| `ALLOWED_PROVIDER_HOSTS` | api.anthropic.com,... | SSRF allow-list |

---

## Ports

| Port | Purpose |
|------|---------|
| 3761 | AI gateway HTTP API (OpenAI-compatible) |

---

## Database Tables

| Table | Purpose |
|-------|---------|
| `ag_key_pool` | Encrypted provider API keys + rotation metadata |
| `ag_key_audit` | Append-only cost/token audit log (90-day retention) |

Migrations: `plugins-pro/paid/nself-ai-gateway/go/migrations/`.

---

## Lanes

Lane affinity controls which keys are eligible for a request. Send
`"x_nself_lane": "<lane>"` in the request body to override auto-detection.

| Lane | Suited for | Example models |
|------|-----------|----------------|
| `fast` | Low-latency / streaming | GPT-4o-mini, Gemini Flash |
| `deep` | Long-context reasoning | Claude Sonnet/Opus |
| `multimodal` | Vision + text | GPT-4o, Gemini Pro Vision |
| `embedding` | Text embeddings | text-embedding-3-large |
| `rerank` | Cross-encoder reranking | TEI rerankers |
| `live` | Real-time sessions | Live API providers |
| `local` | On-premises inference | Ollama, vLLM |

Keys with `lane_affinity = NULL` serve any lane. Keys with a specific
affinity serve only matching lane requests.

---

## Key Pool

### Adding a key

Via the admin API (or Admin UI at `localhost:3021`):

```bash
curl -X POST http://localhost:3761/admin/keys \
  -H "Content-Type: application/json" \
  -H "X-License-Key: $NSELF_PLUGIN_LICENSE_KEY" \
  -d '{
    "provider": "anthropic",
    "key": "sk-ant-api03-...",
    "tier": "shared",
    "quota_day": 1000000,
    "lane_affinity": ["fast", "deep"]
  }'
```

Providers: `anthropic` / `google` / `openai` / `ollama` / `vllm` / `tei`

Tiers: `shared` / `dedicated` / `premium`

### Pool cap

Default 30 keys per provider (set `NSELF_AI_POOL_MAX_KEYS` to adjust).
When full, registration returns HTTP 400.

### Failover and demotion

On 3 consecutive provider errors, a key enters exponential backoff cooldown:

```
cooldown = MIN(2^error_count × 60s, 3600s)
```

A single successful health probe restores the key immediately.

---

## Encryption

Provider API keys are stored encrypted. The envelope format is
`base64(nonce || ciphertext)` (AES-256-GCM, 12-byte random nonce, using
`NSELF_VAULT_KEY` as the 32-byte cipher key).

Security invariants:
- Plaintext keys are never stored, logged, or sent over the wire
- Decryption happens only in-process at request dispatch time
- Audit rows reference keys by UUID only — never by key value
- Vault key rotation requires re-encrypting all pool entries

---

## Tenant Isolation

Dedicated keys (`tenant_id IS NOT NULL`) are reserved for one tenant and take
priority over shared pool keys during selection. The filter
`tenant_id = $caller OR tenant_id IS NULL` is enforced in the selection query
so cross-tenant key access is structurally impossible.

The Hasura row-level filters on `ag_key_pool` and `ag_key_audit` enforce the
same isolation at the GraphQL layer: `tenant_user` role sees only rows matching
`X-Hasura-Tenant-Id`. The `encrypted_key` column is excluded from all
role-based select permissions.

---

## API Reference

### Health check

```
GET http://localhost:3761/health
→ {"status":"ok","service":"nself-ai-gateway"}
```

### List models

```
GET http://localhost:3761/v1/models
→ {"object":"list","data":[{"id":"claude-sonnet-4-5","object":"model","owned_by":"anthropic"}, ...]}
```

### Chat completions

```bash
curl -X POST http://localhost:3761/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-5",
    "messages": [{"role": "user", "content": "Explain key rotation"}],
    "x_nself_lane": "deep"
  }'
```

### Embeddings

```bash
curl -X POST http://localhost:3761/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{"model": "text-embedding-3-large", "input": "text to embed"}'
```

### OpenAI SDK compatibility

```python
import openai

client = openai.OpenAI(
    base_url="http://localhost:3761/v1",
    api_key="not-needed",  # license gate is handled by env var
)
response = client.chat.completions.create(
    model="claude-sonnet-4-5",
    messages=[{"role": "user", "content": "Hello"}],
)
```

---

## Nginx Routes

| Route | Target |
|-------|--------|
| `/ai-gateway/` | AI gateway API (internal routing) |

---

## Verify Install

```bash
nself plugin install nself-ai-gateway
nself status | grep ai-gateway
curl http://localhost:3761/health
```

---

## Troubleshooting

**`402 Payment Required`** — NSELF_PLUGIN_LICENSE_KEY is missing or empty.
Run `nself license set <key>` and restart.

**`NSELF_VAULT_KEY is required`** — Add a 32-byte hex key to your .env:
`echo "NSELF_VAULT_KEY=$(openssl rand -hex 32)" >> .env.dev`

**All keys in cooldown / QUOTA_EXCEEDED** — Check provider error logs:
`nself logs nself-ai-gateway | grep error_count`
Register a new key or wait for cooldown_until to expire.

**`registration_failed: pool at cap`** — Pool is at NSELF_AI_POOL_MAX_KEYS
limit for that provider. Disable unused keys or increase the cap.

---

## Related

- [[Plugin-Overview]] — all available plugins
- [[plugin-ai]] — simpler per-call AI plugin (no pool management)
- [[plugin-claw]] — ɳClaw reasoning plugin (uses this gateway)
- [[Home]]
