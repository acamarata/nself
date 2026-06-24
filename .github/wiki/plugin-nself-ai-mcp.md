# Plugin: nself-ai-mcp

MCP (Model Context Protocol) tool server for nSelf. Exposes 7 tools to local AI
agents (ɳClaw, ClawDE, the nSelf CLI, and plugin sandboxes) over the ADR-003
dispatch chain. Runs on port **3762** with stdio (always active) and SSE
transports. **Pro plugin — requires an active ɳClaw or ClawDE bundle license.**

---

## Overview

`nself-ai-mcp` lets any MCP-compatible AI agent reach the nSelf backend through a
controlled tool surface. It reads provider routes, quota, and key health from
`nself-ai-gateway` (port 3761) and proxies PTY session tooling to `nself-ai-cc`
(port 3760). Every tool call is authorized **before** it runs and the decision is
written to the plugin's own `np_mcp_audit` table.

| Property | Value |
|----------|-------|
| Port | 3762 |
| Transports | stdio (always) + SSE HTTP (`NSELF_MCP_SSE=true`) |
| Bundle | ɳClaw / ClawDE |
| Owned table | `np_mcp_audit` |
| License | required (`requires_license: true`) |

---

## The 7 Tools

| Tool | Tier | Min trust | Mutates | Purpose |
|------|------|-----------|---------|---------|
| `search` | user | standard (2) | no | Hybrid semantic + keyword search over the knowledge base |
| `recall` | user | standard (2) | no | Retrieve a specific memory record by id, topic, or fact hash |
| `summarize` | user | standard (2) | no (LLM call) | Summarize a conversation slice or topic cluster |
| `route` | system | elevated (3) | yes | Select/activate the optimal provider key for a model + tenant |
| `cost-report` | system | elevated (3) | no | Cost breakdown per model/tenant/day from the audit log |
| `key-status` | admin | elevated (3) | no | Key-pool health, error counts, cooldown states (never exposes key material) |
| `safe-write` | user + approval | standard (2) | yes | Write a fact/decision/entity to the knowledge base (requires `user_approved`) |

### search
Hybrid BM25 + vector search with Reciprocal Rank Fusion scoring. Read-only.
```json
{ "tool_id": "search", "input": { "query": "auth decisions", "limit": 10 } }
```

### recall
Retrieve a single memory record by `entity_id`, `topic`, or `fact_hash`.
```json
{ "tool_id": "recall", "input": { "topic": "work.project.alpha" } }
```

### summarize
Summarize a conversation or topic via the gateway key pool. Emits a cost audit row.
```json
{ "tool_id": "summarize", "input": { "source": "topic", "topic": "work.alpha" } }
```

### route
System-tier. Picks the best provider key for a model + tenant from `ag_key_pool`.
```json
{ "tool_id": "route", "input": { "model_id": "claude-sonnet-4-6", "tenant_id": "np_acme" } }
```

### cost-report
System-tier. Aggregates cost from the audit log, grouped by model, day, or caller.
```json
{ "tool_id": "cost-report", "input": { "tenant_id": "np_acme", "group_by": "model" } }
```

### key-status
Admin-only. Returns key health metadata only — never the encrypted key value.
```json
{ "tool_id": "key-status", "input": { "include_cooldown": true } }
```

### safe-write
Writes to the knowledge base. Requires `user_approved: true` in `policy_context`
and a traversal-safe `topic`.
```json
{
  "tool_id": "safe-write",
  "policy_context": { "user_approved": true },
  "input": { "entity_type": "fact", "content": "...", "topic": "work.alpha" }
}
```

---

## Authorization Model (deny-by-default)

Authorization is **deny-by-default**. There is no AllowAll path: a tool runs only
when every gate below passes. Callers are mapped to a trust level by `caller_id`
prefix, and each tool declares the minimum trust it requires.

| Caller pattern | Trust level | Accessible tools |
|----------------|-------------|------------------|
| `clawde:*`, `claude-code:*`, `opencode:*` | elevated (3) | all 7 tools |
| `cli:*` | standard (2) | search, recall, summarize, safe-write |
| `plugin:*` | limited (1) | (none — below standard floor) |
| unauthenticated / unknown prefix | untrusted (0) | none (`PERMISSION_DENIED`) |

- **System-tier tools** (`route`, `cost-report`) require elevated trust (3); a
  `cli:*` caller is denied.
- **Admin-tier tool** (`key-status`) requires elevated trust (3).
- **`safe-write`** additionally requires `user_approved: true`; absence is denied
  before the PolicyEngine even runs.
- A **missing tenant context** is denied for every tool.

---

## Dispatch Chain (ADR-003)

Every invocation traverses six gates in strict order. Any DENY short-circuits with
a typed error and no handler executes.

```
auth → trust → policy → supply-chain → sandbox → handler
```

1. **auth** — resolve trust from `caller_id`; reject unauthenticated callers and
   `safe-write` without approval.
2. **trust** — carry the resolved trust level forward for the audit row.
3. **policy** — `PolicyEngine.evaluate()` performs the per-tool deny-by-default
   check (unknown tool, missing tenant, insufficient trust, missing approval).
   The decision is **always** written to `np_mcp_audit` (ALLOW and DENY).
4. **supply-chain** — `SupplyChainPolicy.check()` validates the requested
   `model_id` for model-selecting tools (`route`, `summarize`) against the model
   allowlist; an unknown model is denied.
5. **sandbox** — `SandboxGuard.check()` validates input. For `safe-write` it
   rejects any **path-traversal** token (`../`, leading `/`, drive letters, NUL,
   `%2e%2e`) in `topic` or `idempotency_key`.
6. **handler** — the tool handler runs and returns its typed result.

---

## Error Taxonomy

| Code | Meaning | Retryable |
|------|---------|-----------|
| `TOOL_NOT_FOUND` | Unknown tool id | no |
| `PERMISSION_DENIED` | Auth/policy gate denied the caller | no |
| `SCHEMA_INVALID` | Input failed schema or sandbox validation | no |
| `POLICY_REJECTED` | PolicyEngine or supply-chain denied | no |
| `BACKEND_UNAVAILABLE` | Gateway/CC backend unreachable | yes (`retry_after_ms`) |
| `QUOTA_EXCEEDED` | Provider quota exhausted | yes (`retry_after_ms`) |

---

## Database — np_mcp_audit

One owned table. One row per PolicyEngine evaluation (ALLOW and DENY).

| Column | Type | Notes |
|--------|------|-------|
| `id` | UUID | primary key |
| `tool_id` | TEXT | tool name |
| `caller_id` | TEXT | caller identity |
| `tenant_id` | TEXT | tenant scope |
| `trace_id` | UUID | distributed trace id |
| `decision` | TEXT | `ALLOW` / `DENY` / `AUDIT_ONLY` |
| `reason` | TEXT | PolicyEngine reason |
| `latency_ms` | INTEGER | evaluation duration |
| `created_at` | TIMESTAMPTZ | row time |
| `source_account_id` | TEXT | multi-app isolation key (default `primary`) |

RLS is enabled and forced. Application rows are filtered by
`app.source_account_id`; Hasura applies the row filter
`{"source_account_id": {"_eq": "X-Hasura-Source-Account-Id"}}` for `nself_user`.
Migrations: `migrations/001_init.sql` + `migrations/002_rls.sql` (with `.down.sql`
counterparts), applied by the nSelf plugin migration runner (`migration_dir: migrations`).

---

## Configuration

| Env var | Required | Default | Purpose |
|---------|----------|---------|---------|
| `NSELF_MCP_TOKEN` | yes | — | Service token; missing → fatal startup |
| `NSELF_DB_URL` | yes | — | PostgreSQL connection string |
| `NSELF_JWT_PUBLIC_KEY` | yes | — | JWT public key for caller validation |
| `NSELF_GATEWAY_URL` | no | `http://localhost:3761` | gateway base URL |
| `NSELF_AICC_URL` | no | `http://localhost:3760` | AI-CC base URL |
| `NSELF_MCP_SSE` | no | `false` | `true` activates the SSE HTTP transport |
| `NSELF_MCP_PORT` | no | `:3762` | SSE listen port |
| `LOG_LEVEL` | no | `info` | `debug` / `info` / `warn` / `error` |

---

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install nself-ai-mcp
nself build
nself start
```

## Health

```
GET http://localhost:3762/health   →  200 OK when the SSE transport is ready
```

---

## License

Source-available. Requires an active ɳClaw or ClawDE bundle license
(`nself license`). The plugin manifest sets `requires_license: true`,
`licenseType: "pro"`, and `requiredEntitlements: ["pro"]`.
