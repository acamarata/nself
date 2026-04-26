# Notify Plugin

> Multi-channel notification dispatch — SMTP, SMS, and push. **Free — MIT licensed.**

## Install

```bash
nself plugin install notify
```

## What It Does

Provides a unified API for sending notifications across SMTP email, SMS, and push channels. Queues notifications and handles retries. Free tier supports SMTP, basic SMS, and push. For A/B testing, scheduled delivery, and user preferences, see [plugin-notify-pro](plugin-notify-pro).

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `NOTIFY_PORT` | `3052` | Notify service port |
| `NOTIFY_SECRET` | *(auto-generated)* | Internal auth secret |
| `NOTIFY_SMTP_HOST` | — | SMTP server hostname |
| `NOTIFY_SMTP_PORT` | `587` | SMTP port |
| `NOTIFY_SMTP_USER` | — | SMTP username |
| `NOTIFY_SMTP_PASS` | — | SMTP password |
| `NOTIFY_SMS_PROVIDER` | — | SMS provider (twilio, etc.) |
| `NOTIFY_FCM_KEY` | — | Firebase Cloud Messaging key |

## Ports

| Port | Purpose |
|------|---------|
| 3052 | Notify service REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_notify_queue` — queued notifications
- `np_notify_log` — delivery history and status

## Nginx Routes

None — notify service is internal only.

## API

```
GET  /health              — Health check
POST /notifications/email — Send email
POST /notifications/sms   — Send SMS
POST /notifications/push  — Send push notification
GET  /notifications       — List notification history
```

## Mobile Push Notifications

For APNs (iOS) and FCM (Android) push delivery, install the dedicated `push` plugin:

```bash
nself plugin install push
```

The `push` plugin provides first-class mobile push delivery with Hasura event-trigger fan-out, delivery state tracking per outbox row, exponential backoff retry, and APNs JWT signing. It is the recommended approach for any consumer app (nChat, nClaw, nFamily, nTV).

See: [[plugin-push]] for full setup, credential management, and Hasura event trigger configuration.

---

[[Home]] | [[Plugin-Overview]] | [[plugin-push]]
