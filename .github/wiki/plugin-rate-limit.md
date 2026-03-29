> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# Rate Limit Plugin

> API rate limiting — per-user, per-endpoint, per-tenant limits with Redis backing. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install rate-limit
```

## What It Does

Applies granular rate limits to your API endpoints. Configure limits per user, per API key, per tenant, or per IP address. Uses Redis for fast in-memory counter storage with sliding window or token bucket algorithms. Requests over the limit receive HTTP 429 with `Retry-After` headers. Limit configurations are managed via REST API without redeploying.

## Dependencies

Requires Redis (`REDIS_ENABLED=true`).

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `RATE_LIMIT_PORT` | `3063` | Rate limit service port |
| `RATE_LIMIT_DEFAULT` | `1000/hr` | Default limit if no rule matches |
| `RATE_LIMIT_ALGORITHM` | `sliding_window` | Algorithm: `sliding_window` or `token_bucket` |
| `RATE_LIMIT_BURST` | `50` | Burst allowance above limit |
| `RATE_LIMIT_KEY_STRATEGY` | `user` | Limit by: `user`, `ip`, `api_key`, `tenant` |

## Ports

| Port | Purpose |
|------|---------|
| 3063 | Rate limit management REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_rate_limit_rules` — limit rule definitions
- `np_rate_limit_violations` — limit exceeded event log

## Nginx Routes

None — rate limiting is enforced at the application layer via middleware.

## API

```
GET  /health          — Health check
GET  /rules           — List all rate limit rules
POST /rules           — Create a rule
PUT  /rules/{id}      — Update a rule
GET  /status/{key}    — Current usage for a key
POST /reset/{key}     — Reset counters for a key
```
