# Webhooks Plugin

> Pro outbound webhook delivery with HMAC-SHA256 signing, exponential-backoff retry, dead-letter queue, and fan-out. Requires a pro license.

## Install

```bash
nself plugin install webhooks
```

## What It Does

Delivers outbound HTTP webhooks to external URLs registered by your users. Signs each payload with HMAC-SHA256 so receivers can verify authenticity. Retries failed deliveries with exponential backoff (5 s, 10 s, 20 s... capped at 30 min). After exhausting retries, moves deliveries to a dead-letter queue for manual inspection and replay. Fan-out: one event fires deliveries to all subscribed endpoints concurrently.

## Security — SSRF Guard

All endpoint URLs are validated at registration time and before every delivery. URLs targeting internal networks are blocked:

- RFC 1918 private ranges: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- Loopback: `127.0.0.0/8`, `::1`
- Link-local: `169.254.0.0/16` (includes `169.254.169.254` AWS/GCP/Azure metadata)
- IPv6 ULA: `fc00::/7`
- Non-http(s) schemes (`file://`, `ftp://`, etc.)

SSRF protection is always on and cannot be disabled. See the [Security-Always-Free Doctrine](https://nself.org/docs/security).

## HMAC Signing

Each delivery includes an HMAC-SHA256 signature:

```
X-Nself-Signature: sha256=<hex>
```

Set a per-endpoint `signing_secret` when registering. Falls back to `WEBHOOKS_SECRET` env var if none is set.

Verify in Python:

```python
import hmac, hashlib

def verify(secret: str, body: bytes, header: str) -> bool:
    expected = "sha256=" + hmac.new(
        secret.encode(), body, hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, header)
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `WEBHOOKS_PORT` | `3087` | Webhooks service port |
| `WEBHOOKS_SECRET` | — | Default HMAC signing secret |
| `WEBHOOKS_MAX_RETRIES` | `8` | Max delivery attempts |
| `WEBHOOKS_TIMEOUT` | `30s` | HTTP delivery timeout |

## Ports

| Port | Purpose |
|------|---------|
| 3087 | Webhooks service REST API (internal only, not exposed via Nginx) |

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_webhook_endpoints` | Registered webhook destinations (URL, signing secret, event subscriptions) |
| `np_webhook_deliveries` | Delivery log: attempt history, status codes, latency |
| `np_webhook_dlq` | Dead-letter queue for deliveries that exhausted retries |

All tables include `source_account_id` for multi-app isolation.

## Nginx Routes

None. The webhooks service is internal only.

## API

```
GET  /health                      Health check
GET  /endpoints                   List registered endpoints
POST /endpoints                   Register an endpoint (SSRF-validated; HTTP 422 on blocked URL)
DELETE /endpoints/{id}            Remove an endpoint
POST /deliver                     Trigger immediate delivery to one endpoint
GET  /deliveries                  Recent delivery history
GET  /deliveries/failed           Dead-letter queue listing
POST /deliveries/{id}/retry       Replay a failed delivery
GET  /metrics                     Prometheus metrics
```

## Register an Endpoint

```bash
curl -X POST http://localhost:3087/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "url": "https://your-app.example.com/webhook",
    "events": ["user.created", "subscription.updated"],
    "signing_secret": "your-hmac-secret"
  }'
```

Returns HTTP 422 for any URL targeting a private or internal address.

---

[[Home]] | [[Plugin-Overview]] | [[cmd-plugin]]
