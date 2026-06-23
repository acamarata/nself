# SMS Plugin

> Twilio SMS messaging with E.164 validation, rate limiting, and opt-out list management. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|-----------------------|
| Free | $0 | $0 | No |
| ɳChat Bundle | $0.99/mo | $9.99/yr | Yes |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** ɳChat Bundle.

## Bundle membership

This plugin is included in the following bundles:

- **ɳChat Bundle** ($0.99/mo or $9.99/yr) — see [[bundle-nchat]]

Or get all bundles and all apps via **ɳSelf+** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install sms
nself build
```

The license is validated against `ping.nself.org/license/validate`. Insufficient tier returns an error with a purchase URL.

## Description

The SMS plugin provides outbound SMS delivery for nSelf applications using Twilio. It handles
phone number validation, per-tenant rate limiting, and opt-out list management in a single
HTTP service on port 9009.

Every outbound message goes through three checks in order:

1. E.164 validation: the `to` number must be in international format (`+14155552671`).
2. Opt-out check: if the number is on the tenant's opt-out list, the send is rejected immediately.
3. Rate limit check: sliding-window counters (per 15-minute window, per hour, per 24h) prevent
   SMS floods. All counters are per-tenant and per-number — one tenant cannot affect another.

Tenant isolation is enforced at two layers: the `tenant_id` column on every `np_sms_*` table,
and Hasura row-level security filters that restrict each tenant to its own rows.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | — | Postgres connection string (required) |
| `SMS_TWILIO_ACCOUNT_SID` | — | Twilio Account SID — starts with `AC` |
| `SMS_TWILIO_AUTH_TOKEN` | — | Twilio Auth Token |
| `SMS_TWILIO_FROM_NUMBER` | — | E.164 sender number purchased on Twilio |
| `SMS_PORT` | `9009` | HTTP listen port |
| `SMS_HOST` | `0.0.0.0` | HTTP listen host |
| `SMS_INTERNAL_SECRET` | — | Optional shared secret for internal calls |
| `SMS_RATE_WINDOW_MINUTES` | `15` | Rate window size in minutes |
| `SMS_RATE_MAX_PER_WINDOW` | `5` | Max sends per window per number |
| `SMS_RATE_MAX_PER_HOUR` | `10` | Max sends per hour per number |
| `SMS_RATE_BLOCK_THRESHOLD_24H` | `50` | Hard block after N sends in 24h |
| `SMS_TEST_MODE` | `false` | Skip Twilio call (for CI and testing) |
| `SMS_PLUGIN_ENABLED` | `true` | Enable or disable the plugin |

## Port

| Service | Port |
|---------|------|
| plugin-sms HTTP | 9009 |

## Database Schema

All tables use the `np_sms_` prefix and include a `tenant_id UUID NOT NULL` column.
Hasura row filter on every table: `{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}`.

| Table | Purpose |
|-------|---------|
| `np_sms_messages` | Outbound message log — Twilio SID, status, error info |
| `np_sms_send_log` | Rate-limit counters per tenant/number/window |
| `np_sms_opt_outs` | Opted-out numbers per tenant (STOP compliance) |

## REST API

All endpoints require the `X-Hasura-Tenant-Id: <uuid>` header (set automatically by Hasura when
your app calls via GraphQL; pass it directly for REST calls).

### POST /sms/send

Send an SMS message.

```bash
curl -X POST http://localhost:9009/sms/send \
  -H "X-Hasura-Tenant-Id: <your-tenant-uuid>" \
  -H "Content-Type: application/json" \
  -d '{"to": "+14155552671", "body": "Your code is 123456"}'
```

Response:

```json
{ "status": "queued", "twilio_sid": "SMxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" }
```

Error responses include an `"error"` field. HTTP 403 means the number has opted out.
HTTP 429 means rate limited.

### POST /sms/opt-out

Add a number to the opt-out list.

```bash
curl -X POST http://localhost:9009/sms/opt-out \
  -H "X-Hasura-Tenant-Id: <uuid>" \
  -H "Content-Type: application/json" \
  -d '{"number": "+14155552671", "source": "reply"}'
```

Sources: `manual`, `reply`, `webhook`.

### DELETE /sms/opt-out

Remove a number from the opt-out list (opt back in).

```bash
curl -X DELETE http://localhost:9009/sms/opt-out \
  -H "X-Hasura-Tenant-Id: <uuid>" \
  -H "Content-Type: application/json" \
  -d '{"number": "+14155552671"}'
```

### GET /health

```bash
curl http://localhost:9009/health
```

Response: `{"plugin":"sms","status":"ok"}`.

## Examples

Send a one-time code:

```bash
curl -X POST http://localhost:9009/sms/send \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"to": "+14155552671", "body": "Your nSelf code is 847291. Valid for 10 minutes."}'
```

Test in CI without real Twilio credentials:

```bash
SMS_TEST_MODE=true SMS_PORT=9009 DATABASE_URL="" nself plugin install sms
```

## Source

Source-available (license required to run): [`plugins-pro/paid/sms/`](https://github.com/nself-org/plugins-pro/tree/main/paid/sms)

`plugins-pro` is a private repository. Source access is granted to ɳSelf+ subscribers and
Enterprise customers.

## See Also

- [[plugin-notify]] — multi-channel notifications (includes SMS via Twilio)
- [[plugin-nself-sms]] — provider-agnostic SMS OTP (Twilio, AWS SNS, Vonage)
- [[bundle-nchat]] — ɳChat bundle that includes this plugin
- [[Plugins]] — full plugin index

← [[Plugins]] | [[Home]] →
