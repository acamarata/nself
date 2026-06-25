# nself-ai-gateway

The nSelf AI Gateway manages a pool of encrypted provider API keys (Anthropic, OpenAI, Google,
Ollama, vLLM, TEI) and dispatches AI requests across them using LRU round-robin rotation with
per-tenant isolation, quota enforcement, and automatic demotion on errors.

**Port:** 3761 | **License:** Pro | **Bundle:** ɳClaw (AI-CP)

---

## Concepts

### Key Pool

The gateway maintains one pool per provider (max 30 keys per pool, configurable via
`NSELF_AI_POOL_MAX_KEYS`). Keys are stored AES-256-GCM encrypted; plaintext is only
decrypted in memory at dispatch time and never logged.

### Lane Affinity

Each key can be restricted to specific capability lanes:

| Lane | Use case |
|------|---------|
| `fast` | Low-latency responses, short context |
| `deep` | Long-context reasoning |
| `multimodal` | Vision / image understanding |
| `embedding` | Text embedding generation |
| `rerank` | Cross-encoder reranking |
| `live` | Real-time streaming |
| `local` | On-premises models (ollama, vLLM) |

Keys with `lane_affinity = NULL` serve all lanes.

### Tenant Isolation

Keys carry an optional `tenant_id`:
- `tenant_id = NULL`: shared pool, accessible to all callers
- `tenant_id = <UUID>`: dedicated to that tenant only

Dedicated keys are preferred over shared pool keys in selection.

---

## Quick Start

```bash
nself license set <your-key>
nself plugin install nself-ai-gateway
```

Set required environment variables:

```bash
nself plugin env set nself-ai-gateway NSELF_VAULT_KEY=<64-hex-char-key>
nself plugin env set nself-ai-gateway NSELF_DB_URL=postgres://...
```

Generate a vault key:

```bash
openssl rand -hex 32
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NSELF_DB_URL` | Yes | — | PostgreSQL connection string |
| `NSELF_VAULT_KEY` | Yes | — | 32-byte AES-256 key as 64-char hex |
| `NSELF_AI_GATEWAY_PORT` | No | `3761` | HTTP server port |
| `NSELF_AI_POOL_MAX_KEYS` | No | `30` | Max keys per provider pool |
| `NSELF_AI_HEALTH_INTERVAL_SEC` | No | `60` | Key health check interval |
| `NSELF_AI_AUDIT_RETENTION_DAYS` | No | `90` | Audit log retention days |
| `NSELF_LICENSE_KEY` | Yes | — | nSelf Pro license key |

---

## API Reference

### Health Check

```
GET /health
→ {"status": "ok"}
→ {"status": "unhealthy"}  (503 if DB unreachable)
```

### Register a Key (Admin)

```
POST /admin/keys
```

```json
{
  "provider": "anthropic",
  "plain_key": "sk-ant-...",
  "tier": "shared",
  "quota_day": 1000,
  "lane_affinity": ["fast", "deep"],
  "tenant_id": null
}
```

`plain_key` is encrypted with AES-256-GCM before storage. Returns `{"id": "uuid"}`.

`provider` must be one of: `anthropic`, `google`, `openai`, `ollama`, `vllm`, `tei`.

`tier` must be one of: `shared`, `dedicated`, `premium`.

Pool cap: at most `NSELF_AI_POOL_MAX_KEYS` enabled keys per provider.

### Dispatch (Select Key)

```
POST /dispatch
```

```json
{
  "provider": "anthropic",
  "lane": "fast",
  "tenant_id": "uuid-or-null"
}
```

Returns:

```json
{
  "key_id": "uuid",
  "provider": "anthropic",
  "key": "sk-ant-...",
  "tier": "shared"
}
```

The `key` field is the decrypted plaintext API key. It is transmitted over TLS only.
Callers must not log or cache this value.

On quota exceeded:

```json
{
  "error": "QUOTA_EXCEEDED",
  "provider": "anthropic",
  "retry_after_ms": 45000
}
```

---

## Key Selection Algorithm

1. Filter: `enabled = true`, `error_count < 3`, `cooldown_until < now()`, lane match, tenant match
2. Sort: dedicated keys first, then LRU (oldest `last_used_at` first)
3. Lock: `FOR UPDATE SKIP LOCKED` for distributed safety
4. Update: decrement `quota_remaining`, set `last_used_at = now()`

---

## AES-256-GCM Encryption

Keys are stored as `base64(nonce || ciphertext)` where:
- `nonce` = 12 random bytes (unique per encryption)
- `ciphertext` = AES-256-GCM(plaintext, NSELF_VAULT_KEY, nonce)

Security invariants:
- Plaintext API key is NEVER stored, NEVER logged
- Decryption occurs only in memory at dispatch time
- `ag_key_audit` stores only the key UUID reference, never key values

---

## Error Demotion

On 3 consecutive errors, a key enters cooldown:

| `error_count` | cooldown |
|---------------|---------|
| 3 | 240s (4 min) |
| 4 | 480s (8 min) |
| 5 | 960s (16 min) |
| 6+ | 3600s (cap) |

Recovery: a single successful health probe clears `error_count = 0` and `cooldown_until = NULL`.

---

## Database Tables

| Table | Purpose |
|-------|---------|
| `ag_key_pool` | Encrypted key registry with quota, demotion, tenant, lane data |
| `ag_key_audit` | Per-call cost audit log (90-day retention) |

`ag_*` tables use `tenant_id UUID` (cloud multi-tenancy), not `source_account_id`.
Hasura row filters enforce tenant isolation on `tenant_admin` role.

---

## License

Requires an active nSelf Pro license. Purchase at [nself.org/pro](https://nself.org/pro).

---

## Related

- [[plugin-nself-ai-mcp]] — MCP dispatch layer over this gateway
- [[Architecture]] — nSelf AI-CP stack overview
- [[Home]]
