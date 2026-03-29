# Notify Pro Plugin

> 5-channel notifications — SMTP, Twilio SMS, FCM Push, Slack, and Telegram. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install notify-pro
```

## What It Does

Extends the free `notify` plugin with a full 5-channel notification system: SMTP email, Twilio SMS, Firebase Cloud Messaging (FCM) push notifications, Slack messages, and Telegram messages. Adds A/B testing for notification content, scheduled delivery windows, per-user channel preferences, and a delivery analytics dashboard.

## Note on Free Tier

The free `notify` plugin provides basic SMTP, SMS, and push dispatch. This pro version adds Slack, Telegram, A/B testing, scheduled delivery, user preferences, and analytics.

## Implementation Details

- **Language:** Rust
- **Port:** 3719
- **Tables:** 2

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `NOTIFY_PRO_PORT` | `3719` | Notify pro service port |
| `NOTIFY_PRO_SMTP_HOST` | — | SMTP server host |
| `NOTIFY_PRO_TWILIO_SID` | — | Twilio Account SID |
| `NOTIFY_PRO_TWILIO_TOKEN` | — | Twilio Auth Token |
| `NOTIFY_PRO_TWILIO_PHONE` | — | Twilio phone number |
| `NOTIFY_PRO_FCM_KEY` | — | Firebase Cloud Messaging server key |
| `NOTIFY_PRO_SLACK_TOKEN` | — | Slack Bot Token |
| `NOTIFY_PRO_TELEGRAM_TOKEN` | — | Telegram Bot API token |

## Ports

| Port | Purpose |
|------|---------|
| 3719 | Notify pro REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_notify_pro_queue` — queued notifications with channel and schedule
- `np_notify_pro_log` — delivery history and A/B test results

## Nginx Routes

None — notify pro is an internal service.
