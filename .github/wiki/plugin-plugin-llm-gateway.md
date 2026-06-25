# plugin-llm-gateway — ClawDE LLM Gateway (ClawDE Bundle)

## Overview

`plugin-llm-gateway` is the ClawDE-specific LLM routing and cost tracking layer.
It proxies LLM requests from the ClawDE daemon to `nself-ai-gateway` (port 3761),
injecting ClawDE session context and tracking per-session token costs in Postgres.

**Port**: 8090 | **Bundle**: ClawDE | **License**: Required

## Architecture

```
ClawDE Daemon → POST /v1/chat
                      ↓
         plugin-llm-gateway (port 8090)
          - Inject clawde_session_id + source_account_id
          - Track cost in np_llm_gw_cost_log
                      ↓
         nself-ai-gateway (port 3761)
                      ↓
         LLM Provider (Anthropic / OpenAI / Gemini / ...)
```

**SSRF guard**: `NSELF_AI_GATEWAY_URL` is set at startup only — no per-request override.

## Installation

```bash
nself plugin install plugin-llm-gateway
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL DSN |
| `NSELF_LICENSE_KEY` | Yes | — | ClawDE or ɳSelf+ license key |
| `NSELF_AI_GATEWAY_URL` | No | `http://localhost:3761` | Upstream gateway (SSRF guard: env only) |
| `PORT` | No | `8090` | Listen port |

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/v1/chat` | Proxy chat with context injection |
| `POST` | `/v1/chat/completions` | OpenAI-compat alias |

### Request Headers

| Header | Description |
|--------|-------------|
| `X-Nself-License` | License key (required) |
| `X-Source-Account-ID` | Tenant identifier |
| `X-Clawde-Session-ID` | ClawDE session for context injection |

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_llm_gw_sessions` | Aggregate token/cost per ClawDE session |
| `np_llm_gw_cost_log` | Per-request cost records with model breakdown |

All tables include `source_account_id TEXT NOT NULL DEFAULT 'primary'` (Convention Wall).

## Security

- **SSRF guard** — upstream gateway URL is env config, never per-request
- **License gate** — every request requires `X-Nself-License` header
- **Tenant isolation** — `source_account_id` on all rows; Hasura RLS enforced
- **No credential forwarding** — ClawDE session IDs injected as metadata only

## Hasura RLS

```yaml
filter:
  source_account_id:
    _eq: X-Hasura-Source-Account-Id
```

## Dependency

Requires `nself-ai-gateway` (port 3761, ticket P4-E4-W1-S02-T03) to be installed and running.

## Docker

```bash
docker run -p 8090:8090 \
  -e DATABASE_URL=postgres://... \
  -e NSELF_LICENSE_KEY=... \
  -e NSELF_AI_GATEWAY_URL=http://nself-ai-gateway:3761 \
  nself/plugin-llm-gateway:latest
```

## Changelog

- **1.0.0** — Initial release (ClawDE bundle, port 8090, context injection + cost tracking)
