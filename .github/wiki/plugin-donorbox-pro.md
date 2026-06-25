# Donorbox Pro Plugin

> Advanced Donorbox donation sync with donor CRM, campaign analytics, and recurring donation management. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install donorbox-pro
```

## What It Does

Extends the free `donorbox` plugin (port 3074) with full webhook processing, a donor CRM layer, campaign performance analytics, and recurring donation lifecycle management. Incoming Donorbox webhooks are verified, stored, and fanned out to downstream tables so your application has a complete local replica of donation activity.

> **Note:** The free `donorbox` plugin (port 3074) handles basic donation sync. This pro version adds CRM, analytics, and recurring donation management on top of that foundation.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DONORBOX_PRO_PORT` | `3005` | Donorbox pro service port |
| `DONORBOX_API_KEY` | — | Donorbox API key |
| `DONORBOX_PLAN_ID` | — | Donorbox campaign/plan ID |
| `DONORBOX_WEBHOOK_SECRET` | — | Secret for verifying incoming Donorbox webhooks |

## Ports

| Port | Purpose |
|------|---------|
| 3005 | Donorbox pro REST API and webhook receiver |

## Database Tables

7 tables added to your Postgres database:
- `np_donorbox_campaigns`, campaign definitions and metadata
- `np_donorbox_donors`, donor CRM records
- `np_donorbox_donations`, individual donation records
- `np_donorbox_recurring`, recurring donation plans and status
- `np_donorbox_analytics`, campaign performance analytics
- `np_donorbox_webhooks`, raw incoming webhook payloads
- `np_donorbox_sync_log`, sync and processing log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/donorbox/` | Donorbox pro API and webhook receiver |

---

See also: [plugin-donorbox](plugin-donorbox) · [Plugin-Overview](Plugin-Overview) · [[Home]]
