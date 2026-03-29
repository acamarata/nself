# Mailgun Plugin

> Mailgun transactional email — an alternative to SendGrid. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install mailgun
```

## What It Does

Integrates Mailgun email delivery into your nSelf backend. Send transactional emails, receive inbound email via routing, handle delivery event webhooks, and manage mailing lists. All activity is logged to Postgres for auditing.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MAILGUN_API_KEY` | — | Mailgun API key |
| `MAILGUN_DOMAIN` | — | Your Mailgun sending domain |
| `MAILGUN_REGION` | `us` | API region: `us` or `eu` |
| `MAILGUN_WEBHOOK_KEY` | — | Webhook signing key |

## Ports

| Port | Purpose |
|------|---------|
| 3040 | Mailgun integration service port |

## Database Tables

4 tables added to your Postgres database:
- `np_mailgun_sends` — email send log
- `np_mailgun_routes` — inbound routing rules
- `np_mailgun_lists` — mailing list sync
- `np_mailgun_events` — delivery event webhook log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/mailgun/webhook` | Mailgun event webhook receiver |
