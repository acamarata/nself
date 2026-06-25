# plugin-email

Transactional email plugin for nSelf. Sends email via [Elastic Email](https://elasticemail.com),
stores message history and delivery events in Postgres, and supports named templates.

**Port:** 9008 | **License:** Pro | **Bundle:** standalone

---

## Quick Start

```bash
nself license set <your-key>
nself plugin install email
```

Set required environment variables, then restart:

```bash
nself plugin env set email ELASTIC_EMAIL_API_KEY=your-key
nself plugin env set email ELASTIC_EMAIL_FROM=noreply@yourdomain.com
nself plugin restart email
```

---

## Elastic Email Setup

1. Create an account at [elasticemail.com](https://elasticemail.com).
2. Go to **Settings → API Keys** and create a key with **Send** permissions.
3. Verify your sending domain under **Settings → Sending Domains**.
4. Copy the API key into `ELASTIC_EMAIL_API_KEY`.

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NSELF_DB_URL` | Yes | — | PostgreSQL connection string |
| `ELASTIC_EMAIL_API_KEY` | Yes | — | Elastic Email API key |
| `ELASTIC_EMAIL_FROM` | Yes | — | Default From address (must be verified) |
| `EMAIL_PLUGIN_PORT` | No | `9008` | HTTP server port |
| `NSELF_LICENSE_KEY` | Yes | — | nSelf Pro license key |

---

## API Reference

All endpoints require `X-Hasura-Source-Account-Id` header (set automatically by nSelf JWT middleware).
Responses are JSON.

### Send Email

```
POST /email/send
```

**Request body:**

```json
{
  "to": "user@example.com",
  "subject": "Hello from nSelf",
  "body_html": "<h1>Hello!</h1>",
  "body_text": "Hello!",
  "template": "welcome"
}
```

`template` is optional. If set, `subject` and `body_html` are loaded from the named template
(per-request values override template values).

**Response:**

```json
{
  "id": "uuid",
  "status": "sent",
  "provider_id": "elastic-message-id"
}
```

On failure: `"status": "failed"` with `"error"` field. Message is still recorded in DB.

---

### List Messages

```
GET /email/messages
```

Returns last 100 messages for the current account, newest first.

```json
[
  {
    "id": "uuid",
    "to": "user@example.com",
    "from": "noreply@yourdomain.com",
    "subject": "Hello",
    "status": "sent",
    "created_at": "2026-06-25T10:00:00Z"
  }
]
```

---

### Templates

#### Create / Update Template

```
POST /email/templates
```

```json
{
  "name": "welcome",
  "subject": "Welcome to {{appName}}",
  "body_html": "<h1>Welcome!</h1>",
  "body_text": "Welcome!"
}
```

Upserts on `(source_account_id, name)`. Returns `{"id": "uuid"}`.

#### List Templates

```
GET /email/templates
```

#### Delete Template

```
DELETE /email/templates/{name}
```

---

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_email_templates` | Named email templates (subject + HTML/text body) |
| `np_email_messages` | All sent/queued/failed messages with provider ID |
| `np_email_events` | Delivery events (delivered, opened, clicked, bounced) |

All tables carry `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation.
Hasura row filters enforce account scoping on all roles.

---

## Message Status Values

| Status | Meaning |
|--------|---------|
| `queued` | Awaiting dispatch |
| `sent` | Accepted by Elastic Email |
| `failed` | Elastic Email returned an error |
| `bounced` | Delivery failed (recorded via webhook) |

---

## Health Check

```
GET /health
→ {"status": "ok"}   (200)
→ {"status": "unhealthy"}  (503 if DB unreachable)
```

---

## License

This plugin requires an active nSelf Pro license. Install with `nself license set <key>`.
Purchase at [nself.org/pro](https://nself.org/pro).
