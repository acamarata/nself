# ɳSelf AI MCP Plugin

> MCP (Model Context Protocol) tool server that exposes 7 nSelf tools to AI agents via the ADR-003 dispatch chain. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|----------------------|
| Free | $0 | $0 | No |
| ɳClaw Bundle | $0.99/mo | $9.99/yr | Yes |
| ClawDE Bundle | $0.99/mo | $9.99/yr | Yes |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** ɳClaw Bundle or ClawDE Bundle ($0.99/mo).

## Bundle membership

This plugin is included in the following bundles:

- **ɳClaw Bundle** ($0.99/mo or $9.99/yr) — see [[bundle-nclaw]]
- **ClawDE Bundle** ($0.99/mo or $9.99/yr) — see [[bundle-clawde]]

Or get all bundles + all apps via **ɳSelf+** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install nself-ai-mcp
nself build
nself start
```

The license is validated against `ping.nself.org/license/validate`.

## Description

The nself-ai-mcp plugin runs an MCP server on port 3762 that gives AI agents
(ClawDE, claude-code, opencode) structured access to nSelf knowledge and controls.

It uses the ADR-003 dispatch chain: every tool call passes through auth, trust
registry, policy engine, supply-chain check, and sandbox guard before the handler
runs. Any step can deny; all PolicyEngine decisions are written to `np_mcp_audit`
for audit trail compliance.

The server speaks two transports: stdio (always active) for direct subprocess
embedding, and SSE HTTP on port 3762 (activated when `NSELF_MCP_SSE=true`) for
network-accessible MCP clients. Provider routes and quota come from
`nself-ai-gateway` (port 3761); PTY session tools proxy to `nself-ai-cc` (port 3760).

## Tools

### search

Hybrid search (BM25 + vector) over the nSelf retrieval index.

- **Caller tier:** user (limited+)
- **Input:** `{ "query": string, "limit": number }`
- **Output:** ranked list of matching documents

```bash
# Via ClawDE agent prompt
Use nself-ai-mcp search: query="plugin install guide" limit=5
```

### recall

Retrieves past session memory fragments for context continuity.

- **Caller tier:** user (limited+)
- **Input:** `{ "session_id": string, "limit": number }`
- **Output:** list of memory fragments from the session

```bash
Use nself-ai-mcp recall: session_id="clawde:abc123" limit=10
```

### summarize

Summarizes content via the gateway's configured LLM provider.

- **Caller tier:** user (limited+)
- **Input:** `{ "content": string }`
- **Output:** summarized text

```bash
Use nself-ai-mcp summarize: content="<long doc text>"
```

### route

Queries the active provider routing table from nself-ai-gateway. Returns which
providers and models are currently available and their priority order.

- **Caller tier:** system (standard+, i.e. `cli:*` and above)
- **Input:** `{}`
- **Output:** provider route list with model, priority, and quota status

```bash
Use nself-ai-mcp route
# Output: [{ provider: "anthropic", model: "claude-...", priority: 1, ... }, ...]
```

### cost-report

Returns usage totals and cost breakdown for the calling account.

- **Caller tier:** system (standard+)
- **Input:** `{ "period": string }` (e.g. `"30d"`, `"7d"`, `"today"`)
- **Output:** per-provider cost breakdown with token counts

```bash
Use nself-ai-mcp cost-report: period="30d"
```

### key-status

Admin tool: returns health and remaining quota for all provider API keys.

- **Caller tier:** admin (elevated only: `clawde:*`, `claude-code:*`, `opencode:*`)
- **Input:** `{}`
- **Output:** per-key status, quota remaining, last-used timestamp

```bash
# Only available to elevated callers (ClawDE, claude-code, opencode sessions)
Use nself-ai-mcp key-status
```

### safe-write

Writes content to nSelf storage. Requires explicit `user_approved: true` in the
policy context. The sandbox guard enforces a non-empty `content` and `topic`.

- **Caller tier:** user (limited+), but ALSO requires `user_approved=true`
- **Input:** `{ "topic": string, "content": string }`
- **Output:** write confirmation with storage path

```bash
Use nself-ai-mcp safe-write: topic="notes/session-summary" content="Today we..."
# Agent must confirm user approval before this tool executes
```

## Auth model

| Caller pattern | Trust level | Accessible tools |
|----------------|-------------|-----------------|
| `clawde:*`, `claude-code:*`, `opencode:*` | elevated (3) | All 7 tools |
| `cli:*` | standard (2) | search, recall, summarize, route, cost-report, safe-write |
| `plugin:*` | limited (1) | search, recall, summarize |
| unauthenticated | untrusted (0) | None |

`key-status` requires elevated trust. `safe-write` requires both
`cli:*` or above AND `user_approved: true`.

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `NSELF_MCP_TOKEN` | yes | — | Auth token; missing = fatal startup |
| `NSELF_DB_URL` | yes | — | PostgreSQL connection string |
| `NSELF_JWT_PUBLIC_KEY` | yes | — | JWT public key for caller validation |
| `NSELF_GATEWAY_URL` | no | `http://localhost:3761` | Gateway base URL |
| `NSELF_AICC_URL` | no | `http://localhost:3760` | AI-CC base URL |
| `NSELF_MCP_SSE` | no | `false` | Set `true` to enable SSE HTTP transport |
| `NSELF_MCP_PORT` | no | `:3762` | SSE listen port |
| `LOG_LEVEL` | no | `info` | Log level: `debug`, `info`, `warn`, `error` |

## Database schema

One table: `np_mcp_audit` — stores one row per PolicyEngine evaluation.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `tool_id` | TEXT | Tool name |
| `caller_id` | TEXT | Caller identity |
| `tenant_id` | TEXT | Tenant scope |
| `trace_id` | UUID | Distributed trace ID |
| `decision` | TEXT | `ALLOW`, `DENY`, or `AUDIT_ONLY` |
| `reason` | TEXT | PolicyEngine reason |
| `latency_ms` | INTEGER | Evaluation duration |
| `created_at` | TIMESTAMPTZ | Row creation time |
| `source_account_id` | TEXT | Multi-app isolation (default: `primary`) |

Row-level security: `source_account_id = app.source_account_id`.
Hasura row filter: `{"source_account_id": {"_eq": "X-Hasura-Source-Account-Id"}}`.

## Ports

| Port | Transport | Notes |
|------|-----------|-------|
| `3762` | HTTP/SSE | Active when `NSELF_MCP_SSE=true` |
| stdio | stdin/stdout | Always active |

Health check: `GET http://localhost:3762/health` returns `200 OK` when ready.

## See also

- [[plugin-nself-ai-gateway]] — provider routing and key management (port 3761)
- [[plugin-nself-ai-cc]] — PTY session proxy (port 3760)
- [[bundle-nclaw]] — ɳClaw bundle including this plugin
- [[bundle-clawde]] — ClawDE bundle including this plugin
- [[Plugin-Overview]] · [[Home]]

← [[Plugin-Overview]] | [[Home]] →
