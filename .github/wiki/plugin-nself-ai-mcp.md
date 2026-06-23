# ɳClaw AI MCP Plugin

> MCP (Model Context Protocol) tool surface for ɳClaw — exposes 7 AI tools to ClawDE, the CLI, and the plugin sandbox via the ADR-003 dispatch chain. **Pro plugin — requires license.**

## Tier Required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|-----------------------|
| Free | $0 | $0 | No |
| ɳClaw Bundle | $0.99/mo | $9.99/yr | Yes |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** ɳClaw bundle or any ɳSelf+ plan (per F07-PRICING-TIERS).

## Bundle Membership

This plugin is included in the following bundles:

- **ɳClaw** ($0.99/mo or $9.99/yr) — see [[bundle-nclaw]]

Or get all bundles via **ɳSelf+** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install nself-ai-mcp
nself build
```

The license is validated against `ping.nself.org/license/validate`. Tier is checked server-side; insufficient tier returns an error.

## What It Does

`nself-ai-mcp` runs on port **3762** and exposes your ɳClaw memory and AI gateway as structured MCP tools. ClawDE, the CLI, and other MCP-compatible clients call these tools instead of building their own Postgres queries or gateway integrations.

Every tool call goes through the ADR-003 dispatch chain:

```
auth → trust → PolicyEngine → SupplyChainPolicy → SandboxGuard → handler
```

Any step returning DENY short-circuits immediately. An audit row is written to `np_mcp_audit` for every PolicyEngine evaluation regardless of decision.

## The 7 Tools

### search

**Permission:** user | **Read-only:** yes

Hybrid semantic + keyword search over the ɳClaw knowledge base using pgvector (semantic) and tsvector (keyword), fused with Reciprocal Rank Fusion (RRF).

```json
{
  "tool_id": "search",
  "caller_id": "clawde:session-abc",
  "tenant_id": "np_tenant_xyz",
  "trace_id": "a1b2c3d4-e5f6-4789-abcd-ef1234567890",
  "timestamp": "2026-06-22T00:00:00.000Z",
  "policy_context": { "user_approved": false },
  "input": {
    "query": "architecture decisions postgres",
    "limit": 10,
    "min_score": 0.1,
    "topic_filter": "work.nself"
  }
}
```

**Output:** `{ results: Array<{ id, content, score, topic, entity_type, created_at }>, total }`

### recall

**Permission:** user | **Read-only:** yes

Retrieve a specific memory record by entity ID, topic path, or SHA-256 fact hash. Exactly one of `entity_id`, `topic`, or `fact_hash` must be provided.

```json
{
  "tool_id": "recall",
  "input": {
    "entity_id": "uuid-v4-here",
    "include_related": true
  }
}
```

**Output:** `{ record: { id, entity_type, content, topic, metadata, created_at, updated_at }, related? }`

### summarize

**Permission:** user | **Side effects:** writes `ag_key_audit` entry

Generate a natural-language summary by calling the LLM gateway (`nself-ai-gateway`). Cost is tracked in `ag_key_audit`.

```json
{
  "tool_id": "summarize",
  "input": {
    "source": "topic",
    "topic": "work.nself.decisions",
    "max_tokens": 500,
    "style": "bullets"
  }
}
```

Supported styles: `brief` (default), `detailed`, `bullets`.

**Output:** `{ summary, model_used, tokens_used, cost_usd }`

### route

**Permission:** system | **Elevated trust required**

Select and activate the optimal provider key from `ag_key_pool` using round-robin with LRU tiebreak. Not accessible to user-tier (cli) callers. System-tier (`clawde:*`, `claude-code:*`, `opencode:*`) required.

```json
{
  "tool_id": "route",
  "input": {
    "model_id": "claude-sonnet-4-6",
    "tenant_id": "np_tenant_xyz",
    "lane": "fast",
    "tokens_estimate": 2000
  }
}
```

Supported lanes: `fast`, `deep`, `multimodal`, `embedding`, `rerank`, `live`, `local`.

**Output:** `{ key_id, provider, quota_remaining, cooldown_until }`

### cost-report

**Permission:** system | **Read-only:** yes | **Elevated trust required**

Aggregate cost data from `ag_key_audit`. Supports grouping by model, day, or caller.

```json
{
  "tool_id": "cost-report",
  "input": {
    "tenant_id": "np_tenant_xyz",
    "start_date": "2026-06-01",
    "end_date": "2026-06-22",
    "group_by": "model"
  }
}
```

**Output:** `{ rows: Array<{ dimension, tokens_used, cost_usd, call_count }>, total_cost_usd }`

### key-status

**Permission:** admin | **Read-only:** yes | **Elevated trust required**

List key pool health, error counts, and cooldown states. Admin-only. Encrypted key values are never returned.

```json
{
  "tool_id": "key-status",
  "input": {
    "provider_filter": "anthropic",
    "include_cooldown": true
  }
}
```

**Output:** `{ keys: Array<{ key_id, provider, tier, enabled, error_count, quota_remaining, cooldown_until, last_used_at }>, pool_size, healthy_count }`

### safe-write

**Permission:** user | **Side effects:** mutates `cb_*` tables | **Requires `user_approved=true`**

Write a memory fact, decision, or entity. Triggers the ingestion pipeline (embedding generation, topic classification, entity extraction).

`policy_context.user_approved = true` is required. Without it, `PERMISSION_DENIED` is returned before any handler logic executes.

```json
{
  "tool_id": "safe-write",
  "policy_context": { "user_approved": true },
  "input": {
    "entity_type": "decision",
    "content": "Use Postgres as the canonical memory store for all ɳClaw facts.",
    "topic": "work.nself.decisions",
    "metadata": { "source": "engineering-review", "confidence": 0.95 },
    "idempotency_key": "decision-postgres-memory-2026-06"
  }
}
```

**Output:** `{ id, status: "created" | "deduplicated", topic_assigned, ingestion_job_id }`

## Auth Model

| Caller Pattern | Trust Level | Can Invoke |
|---|---|---|
| `clawde:*` | 3 (elevated) | All 7 tools |
| `claude-code:*` | 3 (elevated) | All 7 tools |
| `opencode:*` | 3 (elevated) | All 7 tools |
| `cli:*` | 2 (standard) | search, recall, summarize, safe-write |
| `plugin:*` | 1 (limited) | None |
| (unauthenticated) | 0 (untrusted) | None |

## Error Codes

| Code | HTTP | Retry? | Trigger |
|---|---|---|---|
| `TOOL_NOT_FOUND` | 404 | No | tool_id not in registry |
| `PERMISSION_DENIED` | 403 | No | Auth failure; safe-write without user_approved |
| `SCHEMA_INVALID` | 400 | No | Input fails JSON Schema validation |
| `BACKEND_UNAVAILABLE` | 503 | Yes (retry_after_ms) | DB or embedding service unreachable |
| `QUOTA_EXCEEDED` | 429 | Yes (retry_after_ms) | Tenant daily quota exhausted |
| `POLICY_REJECTED` | 403 | No | PolicyEngine returned DENY |

## Configuration

| Env Var | Default | Description |
|---|---|---|
| `NSELF_AI_MCP_PORT` | `3762` | HTTP port |
| `NSELF_AI_MCP_GATEWAY_URL` | `http://localhost:3820` | URL of nself-ai-gateway |
| `NSELF_AI_MCP_DB_URL` | — | Postgres connection string |
| `NSELF_AI_MCP_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `NSELF_LICENSE_KEY` | — | License key (validated at startup) |

## Ports

| Port | Purpose |
|------|---------|
| 3762 | MCP tool HTTP surface |

## Database Tables

| Table | Description |
|---|---|
| `np_mcp_audit` | Audit log of every tool invocation (trace_id, decision, latency_ms) |

Row-level security: `tenant_id` filter for Cloud SaaS; `source_account_id` + `tenant_id IS NULL` for self-host.

The plugin also reads (never writes directly) the following tables owned by `nself-ai-gateway`:

| Table | Accessed by |
|---|---|
| `cb_embeddings` | search, recall |
| `cb_facts` | search, recall, summarize, safe-write |
| `cb_entities` | recall, safe-write |
| `cb_decisions` | recall, safe-write |
| `cb_topics` | search, recall, safe-write |
| `ag_key_pool` | route, key-status |
| `ag_key_audit` | summarize, route, cost-report |

## Health Check

```bash
curl http://localhost:3762/health
```

Returns `200 OK` with `{"status":"ok","version":"0.1.0"}` when the service is healthy.

## Source

`plugins-pro/paid/nself-ai-mcp/` (source-available, license-gated).
Tool catalog and JSON Schema for all 7 tools: `.github/docs/tool-catalog.md`.

## See Also

- [[plugin-nself-ai-gateway]] — LLM key pool and gateway (dependency)
- [[plugin-claw]] — ɳClaw AI assistant engine (primary consumer)
- [[Plugin-Licensing]] — license tiers and pricing
- [[Plugin-Overview]] — all plugins by category
- [[bundle-nclaw]] — ɳClaw bundle contents

---
← [[Plugin-Overview]] | [[Home]] →
