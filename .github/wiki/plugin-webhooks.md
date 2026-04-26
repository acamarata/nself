# Webhooks Plugin

> Outbound webhook dispatcher with retry, HMAC signing, and dead-letter queue. **Free, MIT licensed.**

## Install

```bash
nself plugin install webhooks
```

## What It Does

Delivers outbound HTTP webhooks to external URLs based on Hasura event triggers. Signs payloads with HMAC-SHA256 for verification. Handles retries with exponential backoff, maintains a dead-letter queue for failed deliveries, and logs all attempts.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `WEBHOOKS_PORT` | `3060` | Webhooks service port |
| `WEBHOOKS_SECRET` | — | Default HMAC signing secret |
| `WEBHOOKS_MAX_RETRIES` | `5` | Max delivery attempts |
| `WEBHOOKS_TIMEOUT` | `30s` | HTTP delivery timeout |

## Ports

| Port | Purpose |
|------|---------|
| 3060 | Webhooks service REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_webhooks_endpoints`, registered webhook destinations
- `np_webhooks_deliveries`, delivery log and dead-letter queue

## Nginx Routes

None, webhooks service is internal only.

## API

```
GET  /health                  — Health check
GET  /endpoints               — List webhook endpoints
POST /endpoints               — Register an endpoint
DELETE /endpoints/{id}        — Remove endpoint
POST /deliver                 — Trigger manual delivery
GET  /deliveries              — Delivery history
GET  /deliveries/failed       — Dead-letter queue
POST /deliveries/{id}/retry   — Retry failed delivery
```
