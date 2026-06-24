# ɳNotify Paid Plugin

> Multi-channel notification system with campaigns, push tokens, delivery receipts, DLQ, and webhook dispatch. **ɳClaw + ClawDE + ɳChat bundle plugin.**

> **Requires:** ɳClaw, ClawDE, or ɳChat bundle (or ɳSelf+). `nself license set <key>`

## Install

```bash
nself license set <your-bundle-key>
nself plugin install notify
```

The paid variant runs on port 3712 and is distinct from the core system notify at port 9004.

## What It Does

Provides a full multi-channel notification system: SMTP email, Twilio SMS, Firebase Cloud Messaging (FCM) push, Slack, Discord, Telegram, and outbound webhooks. Adds push notification campaigns with batch dispatch and delivery receipts (DLR), a topic/subscription model for fan-out, an in-app inbox, a dead-letter queue (DLQ) for automatic retry, and CAN-SPAM one-click unsubscribe.

## Implementation Details

- **Language:** Go
- **Port:** 3712
- **Tables:** 15 (`np_notify_*`)

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `NOTIFY_PORT` | `3712` | Plugin service port |
| `NOTIFY_DATABASE_URL` | — | Postgres connection string |
| `NOTIFY_SMTP_HOST` | — | SMTP server hostname |
| `NOTIFY_SMTP_PORT` | `587` | SMTP port |
| `NOTIFY_SMTP_USER` | — | SMTP username |
| `NOTIFY_SMTP_PASS` | — | SMTP password |
| `NOTIFY_SMTP_FROM` | — | Sender email address |
| `NOTIFY_TWILIO_SID` | — | Twilio Account SID |
| `NOTIFY_TWILIO_TOKEN` | — | Twilio Auth Token |
| `NOTIFY_TWILIO_FROM` | — | Twilio phone number |
| `NOTIFY_FCM_KEY` | — | Firebase server key |
| `NOTIFY_SLACK_WEBHOOK` | — | Slack incoming webhook URL |
| `NOTIFY_DISCORD_WEBHOOK` | — | Discord webhook URL |
| `NOTIFY_TELEGRAM_BOT_TOKEN` | — | Telegram Bot API token |
| `NOTIFY_TELEGRAM_CHAT_ID` | — | Telegram chat or channel ID |
| `NOTIFY_UNSUB_SECRET` | — | HMAC secret for unsubscribe tokens (CAN-SPAM) |
| `NOTIFY_UNSUB_BASE_URL` | — | Base URL for unsubscribe links |
| `NOTIFY_POSTAL_ADDRESS` | — | Physical mailing address for email footer |

## Ports

| Port | Purpose |
|------|---------|
| 3712 | Notify paid REST API |

## Database Tables

15 tables added to your Postgres database:

| Table | Purpose |
|-------|---------|
| `np_notify_notifications` | Notification dispatch records |
| `np_notify_webhooks` | Webhook registration and configuration |
| `np_notify_channels` | Named channel configurations |
| `np_notify_templates` | Named notification templates |
| `np_notify_log` | Delivery log records |
| `np_notify_inbox` | In-app notification inbox items |
| `np_notify_retry_queue` | Dead-letter queue for failed dispatches |
| `np_notify_unsubscribes` | CAN-SPAM opt-out registry |
| `np_notify_webpush_subscriptions` | Web Push API subscriptions |
| `np_notify_campaigns` | Push notification campaign builder |
| `np_notify_device_tokens` | FCM and APNs device token registry |
| `np_notify_topics` | Push notification topic registry |
| `np_notify_topic_subscriptions` | Device token subscriptions per topic |
| `np_notify_receipts` | Delivery receipt records per device per campaign |
| `np_notify_config` | Runtime configuration key-value store |

## Key Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/notify` | Send a single notification |
| `POST` | `/notify/bulk` | Send to multiple recipients |
| `POST` | `/notify/template` | Send using a named template |
| `GET`  | `/notify/inbox` | List inbox items for a recipient |
| `POST` | `/notify/inbox/{id}/read` | Mark inbox item as read |
| `POST` | `/notify/inbox/{id}/dismiss` | Dismiss inbox item |
| `POST` | `/notify/campaign` | Create a notification campaign |
| `POST` | `/notify/campaign/{id}/send` | Send a campaign |
| `GET`  | `/notify/receipts` | List delivery receipts |
| `POST` | `/notify/tokens` | Register a device token |
| `GET`  | `/notify/tokens/health` | Token health summary |
| `GET`  | `/notify/topics` | List topics |
| `POST` | `/notify/subscriptions/register` | Register a webhook |
| `GET`  | `/notify/unsubscribe` | CAN-SPAM one-click unsubscribe |

## Verify Installation

```bash
nself plugin status notify
curl http://localhost:3712/health
```

## Difference from Core Notify (Port 9004)

The core system `notify` plugin (port 9004) provides basic dispatch. This paid variant at port 3712 adds:

- Push campaigns with batch dispatch and DLR tracking
- Device token registry and topic fan-out
- In-app inbox with read/dismiss state
- Dead-letter queue with automatic retry
- Webhook registry with per-webhook HMAC signing
- CAN-SPAM unsubscribe management
- Hasura cloud multi-tenancy (tenant_id row-level security)

---

[[Plugin-Overview]] | [[plugin-notify.md]] | [[Home]]
