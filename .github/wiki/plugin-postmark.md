# Postmark Plugin

> Postmark high-deliverability transactional email. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install postmark
```

## What It Does

Integrates Postmark email delivery into your nSelf backend. Postmark specializes in high-deliverability transactional email with fast delivery and detailed bounce management. Supports message streams, template management, and bounce handling. All email events are logged to Postgres.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `POSTMARK_SERVER_TOKEN` | — | Postmark Server API token |
| `POSTMARK_FROM_EMAIL` | — | Default sender email |
| `POSTMARK_STREAM` | `outbound` | Message stream name |
| `POSTMARK_WEBHOOK_TOKEN` | — | Inbound webhook token |

## Ports

| Port | Purpose |
|------|---------|
| 3041 | Postmark integration service port |

## Database Tables

4 tables added to your Postgres database:
- `np_postmark_sends` — email send log
- `np_postmark_bounces` — bounce and complaint records
- `np_postmark_templates` — template sync from Postmark
- `np_postmark_events` — delivery event webhook log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/postmark/webhook` | Postmark inbound webhook |
