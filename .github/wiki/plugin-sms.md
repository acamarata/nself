# plugin-sms

SMS messaging plugin for nSelf, powered by [Twilio](https://www.twilio.com).
Provides SMS send, opt-out management, and delivery tracking.

**Port:** 9009 | **License:** Pro | **Bundle:** ɳChat

---

## Quick Start

```bash
nself license set <your-key>
nself plugin install sms
```

Set required environment variables:

```bash
nself plugin env set sms TWILIO_ACCOUNT_SID=ACxxxxx
nself plugin env set sms TWILIO_AUTH_TOKEN=your-token
nself plugin env set sms TWILIO_FROM_NUMBER=+14155552671
nself plugin restart sms
```

---

## Twilio Setup

1. Create a Twilio account at [twilio.com](https://www.twilio.com).
2. Verify your identity and purchase or provision a phone number.
3. Find your **Account SID** and **Auth Token** in the [Twilio Console](https://console.twilio.com).
4. The phone number must be in **E.164 format** (e.g. `+14155552671`).
5. For production, enable [Programmable Messaging compliance](https://www.twilio.com/en-us/legal/aup).

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NSELF_DB_URL` | Yes | — | PostgreSQL connection string |
| `TWILIO_ACCOUNT_SID` | Yes | — | Twilio Account SID |
| `TWILIO_AUTH_TOKEN` | Yes | — | Twilio Auth Token |
| `TWILIO_FROM_NUMBER` | Yes | — | Sending number (E.164 format) |
| `SMS_PLUGIN_PORT` | No | `9009` | HTTP server port |
| `SMS_RATE_LIMIT_PER_MIN` | No | `10` | Max sends per account per minute |
| `NSELF_LICENSE_KEY` | Yes | — | nSelf Pro license key |

---

## E.164 Phone Number Format

All phone numbers must be in E.164 format: `+<country_code><number>`, with no spaces or dashes.

| Country | Example |
|---------|---------|
| US/Canada | `+14155552671` |
| UK | `+447911123456` |
| Australia | `+61412345678` |

Invalid numbers are rejected with HTTP 400.

---

## API Reference

All endpoints require `X-Hasura-Source-Account-Id` header (set automatically by nSelf JWT middleware).

### Send SMS

```
POST /sms/send
```

```json
{
  "to": "+14155552671",
  "body": "Your verification code is 123456."
}
```

Returns `{"id": "uuid", "status": "sent", "provider_sid": "SM..."}` or
`{"id": "uuid", "status": "failed", "error": "..."}`.

The message is always recorded before dispatch — failures are queryable.

**Rate limit:** Controlled by `SMS_RATE_LIMIT_PER_MIN` (default: 10 per minute per account).
Returns HTTP 429 when exceeded.

**Opt-out check:** If recipient is in opt-out list, returns HTTP 403.

---

### List Messages

```
GET /sms/messages
```

Returns last 100 messages newest-first.

```json
[
  {
    "id": "uuid",
    "to": "+14155552671",
    "from": "+12025551234",
    "body": "Hello!",
    "status": "sent",
    "created_at": "2026-06-25T10:00:00Z"
  }
]
```

---

### Opt-Out Management

#### Add to Opt-Out List

```
POST /sms/opt-out
{"number": "+14155552671"}
```

Returns 204. Future sends to this number return HTTP 403.

#### Remove from Opt-Out List

```
DELETE /sms/opt-out/+14155552671
```

Returns 204.

#### List Opt-Outs

```
GET /sms/opt-outs
```

---

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_sms_messages` | All sent/queued/failed messages |
| `np_sms_opt_outs` | Numbers that have opted out of SMS |

All tables carry `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation.
Hasura row filters enforce `source_account_id = X-Hasura-Source-Account-Id` on all roles.

---

## Security

- Twilio credentials (`TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`) are environment config only.
  They cannot be overridden per request.
- The Twilio API URL is hardcoded in the plugin source — it is not user-configurable (SSRF protection).
- Phone numbers are validated as E.164 before any DB write or Twilio call.
- Opt-out list is checked synchronously before every send.

---

## Message Status Values

| Status | Meaning |
|--------|---------|
| `queued` | Recorded, not yet dispatched |
| `sent` | Accepted by Twilio |
| `failed` | Twilio or config error |
| `delivered` | Confirmed delivered (via Twilio webhook) |
| `undelivered` | Carrier reported non-delivery |

---

## Health Check

```
GET /health
→ {"status": "ok"}      (200)
→ {"status": "unhealthy"}  (503 if DB unreachable)
```

---

## License

Requires an active nSelf Pro license. Install with `nself license set <key>`.
Purchase at [nself.org/pro](https://nself.org/pro).
