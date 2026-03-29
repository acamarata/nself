> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# Twilio Plugin

> SMS and voice call integration via Twilio. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install twilio
```

## What It Does

Integrates Twilio SMS and voice calling into your nSelf backend. Send SMS messages, make outbound calls, receive inbound SMS/calls via webhook, and manage Twilio phone numbers. Logs all communication to Postgres for audit and analytics.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `TWILIO_ACCOUNT_SID` | — | Twilio Account SID |
| `TWILIO_AUTH_TOKEN` | — | Twilio Auth Token |
| `TWILIO_PHONE_NUMBER` | — | Your Twilio phone number |
| `TWILIO_WEBHOOK_SECRET` | — | Webhook signature validation secret |

## Ports

| Port | Purpose |
|------|---------|
| 3038 | Twilio integration service port |

## Database Tables

4 tables added to your Postgres database:
- `np_twilio_messages` — SMS send/receive log
- `np_twilio_calls` — voice call records
- `np_twilio_phone_numbers` — managed phone numbers
- `np_twilio_webhook_events` — inbound webhook log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/twilio/webhook` | Twilio inbound webhook receiver |
