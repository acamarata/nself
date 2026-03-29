# SendGrid Plugin

> SendGrid transactional and marketing email with template management. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install sendgrid
```

## What It Does

Integrates SendGrid email delivery into your nSelf backend. Send transactional emails via templates, manage marketing lists, handle inbound email parsing, and receive delivery event webhooks. All email activity is logged to Postgres.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SENDGRID_API_KEY` | — | SendGrid API key |
| `SENDGRID_FROM_EMAIL` | — | Default sender email address |
| `SENDGRID_FROM_NAME` | — | Default sender name |
| `SENDGRID_WEBHOOK_KEY` | — | Webhook verification key |

## Ports

| Port | Purpose |
|------|---------|
| 3039 | SendGrid integration service port |

## Database Tables

4 tables added to your Postgres database:
- `np_sendgrid_sends` — email send log
- `np_sendgrid_templates` — template sync from SendGrid
- `np_sendgrid_contacts` — contact list sync
- `np_sendgrid_events` — delivery event webhook log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/sendgrid/webhook` | SendGrid event webhook receiver |
