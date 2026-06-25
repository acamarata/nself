# nself-ai-mcp

The nSelf AI MCP server exposes 7 tools to local AI agents (ClawDE, Claude Code, OpenCode,
CLI) over the ADR-003 6-step dispatch chain. Every tool invocation is auth-gated by caller
trust level before reaching the PolicyEngine.

**Port:** 3762 | **License:** Pro | **Bundle:** ɳClaw (AI-CP)

---

## 7 Tools

| Tool | Trust required | Description |
|------|---------------|-------------|
| `search` | standard (2) | Semantic search over plugin, task, and user data |
| `recall` | standard (2) | Recall prior context from the nSelf memory index |
| `summarize` | standard (2) | Summarize a nSelf resource by ID or topic |
| `safe-write` | standard (2) + user_approved | Write structured output to a scoped memory topic |
| `route` | elevated (3) | Route a request to the optimal AI provider and lane |
| `cost-report` | elevated (3) | Pull token cost summary from ag_key_audit |
| `key-status` | elevated (3) + admin grant | Show key pool health per provider |

---

## Dispatch Chain (ADR-003)

Every invocation runs 6 ordered steps. Any failure short-circuits — the handler never executes.

```
caller → auth → trust → PolicyEngine → SupplyChainPolicy → SandboxGuard → handler
```

| Step | Component | Action |
|------|-----------|--------|
| 1 | Auth | Resolve caller_id to trust level; deny untrusted immediately |
| 2 | Trust | Pass trust context through chain |
| 3 | PolicyEngine | Evaluate policy (W4-W7: pass-through; W14-T02: deny-by-default) |
| 4 | SupplyChainPolicy | Validate model allowlist and MCP server binary hashes |
| 5 | SandboxGuard | Per-tool input validation |
| 6 | Handler | Invoke the registered tool handler |

Every PolicyEngine evaluation writes a row to `ag_key_audit` (ALLOW and DENY alike).

---

## Trust Model

Caller identity is determined by the `caller_id` prefix in the MCP envelope.

| Caller prefix | Trust level | Notes |
|--------------|-------------|-------|
| `clawde:*` | 3 (elevated) | LEDGER §C lock |
| `claude-code:*` | 3 (elevated) | LEDGER §C lock |
| `opencode:*` | 3 (elevated) | LEDGER §C lock |
| `cli:*` | 2 (standard) | Standard operations |
| `plugin:*` | 1 (limited) | Plugin sandbox |
| (unmatched) | 0 (untrusted) | All tools denied |

Trust level 3 (elevated) can access all 7 tools. Trust level 2 (standard) reaches user-tier
tools only (search, recall, summarize, safe-write). Trust levels 0-1 are denied all tools.

---

## Tool Reference

### search

Semantic search over nSelf plugin metadata, task records, and user data.

**Inputs:**

```json
{
  "query": "string (required)",
  "limit": "integer (optional, default 10)",
  "scope": "plugins | tasks | users (optional, default all)"
}
```

**Requires:** trust_level >= 2

---

### recall

Recall prior conversation context or topic summaries from the memory index.

**Inputs:**

```json
{
  "topic": "string (required)",
  "max_age_days": "integer (optional, default 30)"
}
```

**Requires:** trust_level >= 2

---

### summarize

Summarize a nSelf resource (plugin detail, task history, user profile).

**Inputs:**

```json
{
  "resource_id": "string (required)",
  "resource_type": "plugin | task | user (required)"
}
```

**Requires:** trust_level >= 2

---

### safe-write

Write structured output to a scoped memory topic. Requires explicit user approval
in the policy context to prevent unintended writes.

**Inputs:**

```json
{
  "topic": "string (required)",
  "content": "string (required, non-empty)"
}
```

**Requires:** trust_level >= 2 AND `policy_context.user_approved = true`

If `user_approved` is false, the request is denied before PolicyEngine runs.

---

### route

Route an AI request to the optimal provider and lane via nself-ai-gateway.

**Inputs:**

```json
{
  "provider": "anthropic | openai | google | ollama | vllm | tei",
  "lane": "fast | deep | multimodal | embedding | rerank | live | local",
  "tenant_id": "uuid or null"
}
```

**Requires:** trust_level = 3 (elevated callers only)

---

### cost-report

Pull token cost summary from `ag_key_audit` for the current tenant.

**Inputs:**

```json
{
  "period_days": "integer (optional, default 30)",
  "provider": "string (optional, filter by provider)"
}
```

**Requires:** trust_level = 3 (elevated callers only)

---

### key-status

Show current key pool health (enabled count, cooldown count, quota remaining) per provider.

**Inputs:**

```json
{
  "provider": "string (optional, show all if omitted)"
}
```

**Requires:** trust_level = 3 AND admin grant (admin-only tool)

---

## Quick Start

```bash
nself license set <your-key>
nself plugin install nself-ai-mcp
```

Set required environment variables:

```bash
nself plugin env set nself-ai-mcp NSELF_LICENSE_KEY=<your-key>
nself plugin env set nself-ai-mcp NSELF_AI_GATEWAY_URL=http://localhost:3761
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NSELF_LICENSE_KEY` | Yes | — | nSelf Pro license key |
| `NSELF_MCP_PORT` | No | `3762` | HTTP server port |
| `NSELF_AI_GATEWAY_URL` | No | `http://localhost:3761` | nself-ai-gateway endpoint |
| `NSELF_DB_URL` | No | — | PostgreSQL for audit writes (optional) |

---

## Database

`nself-ai-mcp` writes to `ag_key_audit` (owned by nself-ai-gateway) but maintains no
tables of its own. `plugin.json` declares `"tables": []`.

Tenant isolation is inherited from nself-ai-gateway's `ag_key_audit` row-filter on
`tenant_id` (cloud multi-tenancy, not source_account_id).

---

## License

Requires an active nSelf Pro license. Purchase at [nself.org/pro](https://nself.org/pro).

---

## Related

- [[plugin-nself-ai-gateway]] — AI key pool that this plugin dispatches through
- [[plugin-retrieval]] — pgvector retrieval backing the search tool
- [[Architecture]]
- [[Home]]
