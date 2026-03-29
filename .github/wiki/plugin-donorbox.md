# Donorbox Plugin

> Donation data sync from Donorbox with webhook support. **Free — MIT licensed.**

## Install

```bash
nself plugin install donorbox
```

## What It Does

Syncs donation and donor data from Donorbox into your Postgres database in real time via webhooks. Tracks campaigns, donors, one-time and recurring donations, and refunds. The free tier provides core sync functionality. See [plugin-donorbox-pro](plugin-donorbox-pro) for advanced reporting and CRM integrations.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DONORBOX_PORT` | `3074` | Service port |
| `DONORBOX_WEBHOOK_SECRET` | — | Donorbox webhook secret key |
| `DONORBOX_API_KEY` | — | Donorbox API key for initial sync |

## Ports

| Port | Purpose |
|------|---------|
| 3074 | Donorbox webhook receiver and API |

## Database Tables

7 tables added to your Postgres database:
- `np_donorbox_campaigns` — donation campaigns
- `np_donorbox_donors` — donor profiles
- `np_donorbox_donations` — donation records
- `np_donorbox_recurring_plans` — recurring donation plans
- `np_donorbox_refunds` — refund records
- `np_donorbox_questions` — custom campaign questions
- `np_donorbox_webhook_events` — raw webhook event log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/donorbox/webhook` | Donorbox webhook receiver |
