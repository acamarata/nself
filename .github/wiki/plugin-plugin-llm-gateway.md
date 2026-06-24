# plugin-llm-gateway

ClawDE LLM gateway — routes ClawDE AI requests through `nself-ai-gateway` with
session context injection, per-session cost tracking, and JWT-verified tenant
isolation. Part of the **ClawDE bundle** (port 8090).

## Quick Start

```bash
# Install (requires ClawDE bundle license)
nself plugin install plugin-llm-gateway

# Verify it is running
curl http://localhost:8090/healthz
# {"status":"ok"}
```

## What It Does

`plugin-llm-gateway` is a consumer plugin layered on top of `nself-ai-gateway`
(port 3761). It adds three ClawDE-specific capabilities:

1. **Session context injection** — before forwarding a request, the gateway
   retrieves the stored ClawDE session context blob from `np_llm_gw_sessions` and
   prepends it as a `system` message. This gives the LLM workspace and thread
   awareness across calls without the client needing to re-send context each time.

2. **Cost tracking** — after every successful completion, an immutable row is
   appended to `np_llm_gw_cost_log` with token counts and an estimated USD cost.
   This lets ClawDE surface per-session and per-user cost summaries.

3. **JWT-verified identity isolation** — every request must carry a valid RS256 JWT.
   The `user_id` used in all database queries is derived from the verified `sub` claim
   only. No client-supplied header (such as `X-Hasura-Tenant-Id`) is ever used as
   an identity source.

## Requirements

| Requirement | Notes |
|---|---|
| nSelf version | >= 1.1.1 |
| Bundle | ClawDE (or ɳSelf+) |
| Depends on | `nself-ai-gateway` (port 3761) — must be installed first |
| License | Required (`nself license activate`) |

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `NSELF_AI_GATEWAY_URL` | Yes | Upstream URL (e.g. `http://localhost:3761`). Never set per-request. |
| `NSELF_DB_URL` | Yes | Postgres connection string. |
| `NSELF_JWT_PUBLIC_KEY` | Yes | RS256 PEM public key for JWT verification. |
| `NSELF_PLUGIN_LICENSE_KEY` | Yes | License key (ClawDE or ɳSelf+ tier). |
| `NSELF_LLM_GATEWAY_PORT` | No | Listen port (default: `8090`). |

`nself` injects all required variables automatically when started via `nself start`.

## Endpoints

### `POST /v1/chat/completions`

Forward a ClawDE AI request to `nself-ai-gateway` with session context injected.

**Headers:**

| Header | Required | Notes |
|---|---|---|
| `Authorization` | Yes | `Bearer <jwt>` — RS256 JWT issued by nSelf auth service |
| `X-Nself-License-Key` | Yes | Injected automatically by nSelf runtime |
| `ClawDE-Session-Id` | No | Session identifier for context injection and cost logging |
| `Content-Type` | Yes | `application/json` |

**Body:** Standard OpenAI chat completions request.

```json
{
  "model": "claude-sonnet-4-5",
  "messages": [
    {"role": "user", "content": "What is the current file?"}
  ]
}
```

**Response:** Proxied response from `nself-ai-gateway`.

**Error responses:**

| Status | Code | Meaning |
|---|---|---|
| 401 | `auth_failed` | Missing or invalid JWT |
| 402 | `license_invalid` | Missing or invalid license key |
| 400 | `bad_request` | Malformed request body |
| 502 | `upstream_error` | `nself-ai-gateway` unreachable |

### `GET /v1/sessions/{sessionID}`

Retrieve the stored context JSON for a session, scoped to the authenticated user.

**Headers:** `Authorization: Bearer <jwt>`

**Response:**

```json
{
  "session_id": "workspace-abc:thread-123",
  "context_json": { ... },
  "model": "claude-sonnet-4-5",
  "provider": "anthropic"
}
```

Returns `404` if the session does not exist for the authenticated user.

### `GET /healthz`

Liveness probe. No authentication required.

```json
{"status":"ok"}
```

## Security

### Identity Trust Chain

```
ClawDE client
  → Authorization: Bearer <jwt>
    → plugin-llm-gateway verifies RS256 signature (NSELF_JWT_PUBLIC_KEY)
      → user_id = jwt.sub  (ONLY trusted source)
        → DB queries scoped to user_id + source_account_id
```

`X-Hasura-Tenant-Id`, `X-Hasura-User-Id`, and similar headers are **never used**
as identity sources. A request that supplies only these headers receives `401`.

### SSRF Protection

The upstream `nself-ai-gateway` URL is read once from `NSELF_AI_GATEWAY_URL` at
process startup. It must be an `http://` or `https://` URL. Clients cannot specify
or override the upstream URL in any request header or body field.

### Multi-App Isolation

All database queries use `source_account_id` (Multi-App Isolation column) to scope
data per-account within a single nSelf deployment. This is the correct P4 isolation
mechanism. `tenant_id` (Cloud Multi-Tenancy) is not used in this plugin.

### Append-Only Cost Log

`np_llm_gw_cost_log` is written by the Go service only (as `nself_admin`). The
`user` role has `SELECT` access via Hasura but no `INSERT`, `UPDATE`, or `DELETE`
permissions. This makes the cost log tamper-evident: a user cannot inflate or erase
their own cost history.

## Database Tables

### np_llm_gw_sessions

Stores ClawDE session context per user. Rows are upserted on context updates.

```sql
CREATE TABLE np_llm_gw_sessions (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id        TEXT        NOT NULL,
  user_id           UUID        NOT NULL,
  context_json      JSONB       NOT NULL DEFAULT '{}',
  model             TEXT        NOT NULL DEFAULT '',
  provider          TEXT        NOT NULL DEFAULT '',
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  source_account_id TEXT        NOT NULL DEFAULT 'primary',
  CONSTRAINT uq_llm_gw_sessions_user_session
    UNIQUE (user_id, session_id, source_account_id)
);
```

### np_llm_gw_cost_log

Append-only cost record per LLM request.

```sql
CREATE TABLE np_llm_gw_cost_log (
  id                UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id        TEXT           NOT NULL,
  user_id           UUID           NOT NULL,
  model             TEXT           NOT NULL,
  provider          TEXT           NOT NULL DEFAULT '',
  prompt_tokens     INT            NOT NULL DEFAULT 0,
  completion_tokens INT            NOT NULL DEFAULT 0,
  total_tokens      INT            NOT NULL DEFAULT 0,
  cost_usd          NUMERIC(12, 8) NOT NULL DEFAULT 0,
  request_id        UUID           NOT NULL DEFAULT gen_random_uuid(),
  created_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
  source_account_id TEXT           NOT NULL DEFAULT 'primary'
);
```

## Hasura Permissions

Both tables use the `user` / `nself_admin` two-role model:

- **`user` role** — `SELECT` only, filtered by
  `source_account_id _eq X-Hasura-Source-Account-Id`.
- **`nself_admin` role** — full access (INSERT, UPDATE, DELETE where needed).
- **`np_llm_gw_cost_log`** — no `INSERT` for `user` role via GraphQL; writes happen
  via the Go service under `nself_admin`.

`user` insert on `np_llm_gw_sessions` sets `source_account_id` and `user_id` via
Hasura preset columns — clients cannot supply arbitrary values for these fields.

## Observability

Cost data is available via GraphQL (for `user` role) or direct SQL (for admin):

```graphql
query MyCosts {
  np_llm_gw_cost_log(
    order_by: { created_at: desc }
    limit: 100
  ) {
    session_id
    model
    total_tokens
    cost_usd
    created_at
  }
}
```

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `401 auth_failed` | JWT missing or expired | Check `Authorization` header; renew JWT |
| `402 license_invalid` | License key not set | `nself license activate` with ClawDE key |
| `502 upstream_error` | `nself-ai-gateway` down | `nself plugin status nself-ai-gateway` |
| `404` from `/v1/sessions/{id}` | Session not found for this user | Correct `ClawDE-Session-Id` or user mismatch |

## Related

- [nself-ai-gateway](plugin-nself-ai-gateway.md) — upstream LLM router (port 3761)
- [plugin-pty](plugin-plugin-pty.md) — ClawDE terminal session support
- [ClawDE bundle](bundle-clawde.md) — full ClawDE feature set

## SPORT

- `F04-PLUGIN-INVENTORY-PRO.md` — plugin row
- `F10-PORT-REGISTRY.md` — port 8090

## License

Source-Available. Part of `nself-org/plugins-pro`. Requires an active ClawDE
bundle or ɳSelf+ license. Free plugins and MIT app repos do not include this plugin.
