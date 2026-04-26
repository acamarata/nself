# Notifications Plugin

> In-app notification system with templates, delivery tracking, and GraphQL subscriptions. **Free, MIT licensed.**

## Install

```bash
nself plugin install notifications
```

## What It Does

Adds a complete in-app notification system to your Hasura backend. Supports multi-channel delivery (email, push, SMS) with template management and per-channel delivery tracking. Exposes both a REST API and GraphQL subscriptions for real-time notification updates.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `NOTIFICATIONS_PORT` | `3102` | Notifications service port |
| `NOTIFICATIONS_SMTP_HOST` | — | SMTP host for email channel |
| `NOTIFICATIONS_SMTP_PORT` | `587` | SMTP port |
| `NOTIFICATIONS_FCM_KEY` | — | Firebase key for push |

## Ports

| Port | Purpose |
|------|---------|
| 3102 | Notifications REST API |

## Database Tables

3 tables added to your Postgres database:
- `np_notifications_notifications`, notification records
- `np_notifications_templates`, message templates per channel
- `np_notifications_preferences`, per-user channel preferences

## Nginx Routes

| Route | Target |
|-------|--------|
| `/notifications/` | Notifications API |

## GraphQL

Tables are automatically tracked in Hasura. Subscribe to `np_notifications_notifications` for real-time updates using Hasura's subscription API.
