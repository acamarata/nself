# SMS Plugin

> Twilio SMS delivery, OTP codes, and status webhooks. **ɳChat bundle — paid.**

## Install

```bash
nself plugin install sms
```

Requires an active [ɳChat bundle](https://nself.org/products/nchat) subscription or ɳSelf+.

## What It Does

The SMS plugin exposes a license-gated HTTP service on port **9009** that:

- Sends SMS messages via Twilio (E.164 validated, SSRF-guarded).
- Generates and delivers 6-digit OTP codes for phone-number verification.
- Receives Twilio status-callback webhooks to track delivery state.
- Enforces per-tenant per-number rate limiting (1 SMS / minute default).
- Logs all outbound messages to `np_sms_messages` in your nSelf Postgres instance.

## Prerequisites

You need a [Twilio account](https://www.twilio.com/try-twilio) with:

- Account SID (starts with `AC`)
- Auth Token
- A Twilio phone number with SMS capability

## Configuration

Set these environment variables (via `nself env set` or your `.env` file):

```bash
nself env set SMS_TWILIO_ACCOUNT_SID=ACxxxxxxxxxx
nself env set SMS_TWILIO_AUTH_TOKEN=your_token
nself env set SMS_TWILIO_FROM_NUMBER=+15551234567
```

| Variable | Required | Description |
|----------|----------|-------------|
| `TWILIO_ACCOUNT_SID` | Yes | Twilio Account SID |
| `TWILIO_AUTH_TOKEN` | Yes | Twilio Auth Token |
| `TWILIO_FROM_NUMBER` | Yes | Your Twilio phone number (E.164) |
| `DATABASE_URL` | No | Postgres connection string (enables message logging) |
| `NSELF_LICENSE_KEY` | Auto | Set by `nself plugin install sms` |
| `PORT` | No | Listen port (default: `9009`) |
| `LOG_LEVEL` | No | `debug`, `info`, `warn`, `error` (default: `info`) |

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | None | Health check |
| `POST` | `/sms/send` | License | Send an SMS |
| `POST` | `/sms/otp/send` | License | Send a 6-digit OTP |
| `GET` | `/sms/messages` | License | List sent messages |
| `POST` | `/sms/webhook` | License | Twilio status callback |

## Ports

| Port | Purpose |
|------|---------|
| 9009 | SMS service REST API |

## Usage Examples

### Send an SMS

```bash
curl -X POST http://localhost:9009/sms/send \
  -H "Content-Type: application/json" \
  -H "X-Nself-License: $NSELF_LICENSE_KEY" \
  -H "X-Hasura-Tenant-Id: $TENANT_ID" \
  -d '{"to":"+15551234567","body":"Hello from ɳSelf!"}'
```

**Response:**
```json
{
  "sid": "SMxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "status": "queued",
  "to": "+15551234567"
}
```

### Send an OTP

```bash
curl -X POST http://localhost:9009/sms/otp/send \
  -H "Content-Type: application/json" \
  -H "X-Nself-License: $NSELF_LICENSE_KEY" \
  -H "X-Hasura-Tenant-Id: $TENANT_ID" \
  -d '{"to":"+15551234567"}'
```

The user receives: `"Your verification code: 847291"`

### Check health

```bash
curl http://localhost:9009/health
# {"status":"ok","service":"plugin-sms","port":"9009"}
```

## Database Tables

| Table | Description |
|-------|-------------|
| `np_sms_messages` | Log of every sent SMS with Twilio SID, status, and timestamps |
| `np_sms_rate_limits` | Per-tenant per-number rate limit state |
| `np_sms_otp_codes` | OTP codes with expiry and used-at tracking |

All tables include `tenant_id` (cloud multi-tenancy) and `source_account_id` (multi-app isolation). Hasura RLS row filters are applied to all tables.

## Phone Number Format

All phone numbers must be in **E.164 format**: `+` followed by country code and number, no spaces or dashes. Examples:

- `+15551234567` (US)
- `+447911123456` (UK)
- `+919876543210` (India)

Invalid numbers return HTTP 400 before any Twilio API call is made.

## Security

| Control | Details |
|---------|---------|
| **License gate** | All endpoints (except `/health`) require `X-Nself-License` header |
| **SSRF guard** | Only `api.twilio.com` is reachable; RFC-1918 + loopback blocked at transport |
| **E.164 validation** | Numbers validated before any outbound call |
| **Rate limiting** | 1 SMS / minute per (tenant, destination number) |
| **Hasura RLS** | `tenant_id` + `source_account_id` row filters on all tables |

The SSRF guard operates at the HTTP transport layer — even internal services cannot trick the plugin into calling RFC-1918 addresses.

## Error Reference

| HTTP Status | Code | Meaning |
|-------------|------|---------|
| 400 | `invalid_e164` | Phone number not in E.164 format |
| 400 | `missing_fields` | Required `to` or `body` missing |
| 401 | `license_required` | `X-Nself-License` header absent |
| 403 | `license_invalid` | License key rejected |
| 403 | `ssrf_blocked` | Outbound request blocked by SSRF guard |
| 429 | `rate_limited` | 1 SMS/min limit exceeded |
| 503 | `no_twilio` | Twilio credentials not configured |

## Twilio Webhooks

Set your Twilio number's **Status Callback URL** to:

```
https://your-nself-domain/plugins/sms/webhook
```

The plugin logs `MessageSid` and `MessageStatus` from each callback.

## Docker

```bash
# Pull
docker pull nself/plugin-sms:latest

# Run manually (for testing)
docker run \
  -e TWILIO_ACCOUNT_SID=ACxx \
  -e TWILIO_AUTH_TOKEN=token \
  -e TWILIO_FROM_NUMBER=+15550000000 \
  -e NSELF_LICENSE_KEY=your_key \
  -p 9009:9009 \
  nself/plugin-sms:latest
```

## Uninstall

```bash
nself plugin uninstall sms
```

This stops and removes the plugin container. Database tables (`np_sms_*`) are preserved and must be dropped manually if you want a full cleanup.

---

[[Home]] | [[Plugin-Overview]] | [[plugin-notify]] | [[plugin-push]]
