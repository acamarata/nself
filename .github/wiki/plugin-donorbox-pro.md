# Donorbox Plugin

> Real-time Donorbox donation sync with multi-account support, webhook processing, and donor analytics. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|----------------------|
| Free | $0 | $0 | No |
| Any bundle | $0.99/mo | $9.99/yr | If in bundle |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** Basic (this is a `tier: pro` plugin per F07-PRICING-TIERS).

## Bundle membership

This plugin is not currently in a named bundle. A Basic-tier subscription or higher unlocks it.

Or get all bundles + all apps via **ɳSelf+** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install donorbox
nself build
```

The license is validated against `ping.nself.org/license/validate`. Tier is checked server-side; insufficient tier returns an error.

## Description

The donorbox pro plugin (port 3005) syncs campaigns, donors, one-time donations, recurring plans, and ticketing data from Donorbox into your Postgres database. Syncs run on a configurable schedule and a real-time webhook endpoint keeps records current with every incoming Donorbox event.

Multi-account support lets a single nSelf project connect to multiple Donorbox accounts. Each account's data is isolated via `source_account_id` so queries and analytics stay scoped per account. Configure additional accounts with `DONORBOX_EMAILS` and `DONORBOX_API_KEYS` as comma-separated lists.

**Distinct from the free donorbox plugin (port 3074):** The free plugin provides basic webhook receipt and sync. This pro plugin adds multi-account isolation, configurable sync intervals, donor analytics tables, event tracking, and ticketing data. Both plugins can coexist; they run on separate ports and write to separate table sets.

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | Postgres connection string (auto-provided by nSelf) |
| `DONORBOX_API_KEY` | Yes | — | Donorbox API key for primary account |
| `DONORBOX_EMAIL` | Yes | — | Donorbox account email for primary account |
| `DONORBOX_WEBHOOK_SECRET` | No | — | Secret for verifying incoming Donorbox webhooks |
| `DONORBOX_EMAILS` | No | — | Comma-separated emails for multi-account setup |
| `DONORBOX_API_KEYS` | No | — | Comma-separated API keys for multi-account setup |
| `DONORBOX_ACCOUNT_LABELS` | No | — | Human-readable labels for each account (comma-separated) |
| `DONORBOX_WEBHOOK_SECRETS` | No | — | Per-account webhook secrets (comma-separated) |
| `DONORBOX_SYNC_INTERVAL` | No | `3600` | Sync interval in seconds |

Reference vault credentials. Never hardcode secrets.

## Ports

| Port | Purpose |
|------|---------|
| 3005 | Donorbox pro REST API and webhook receiver |

Bound to `127.0.0.1`; access via Nginx, never directly.

## Database Schema

Tables created (prefix `np_`):

| Table | Contents |
|-------|----------|
| `np_donorbox_campaigns` | Campaign definitions and metadata |
| `np_donorbox_donors` | Donor profiles and contact records |
| `np_donorbox_donations` | Individual donation records |
| `np_donorbox_plans` | Recurring donation plans |
| `np_donorbox_events` | Donor lifecycle events |
| `np_donorbox_tickets` | Campaign ticketing records |
| `np_donorbox_webhook_events` | Raw incoming webhook payload log |

All tables include `source_account_id` for multi-account isolation.

## REST API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness probe |
| GET | `/` | Plugin capability list |
| POST | `/webhook` | Donorbox webhook receiver (HMAC-verified) |
| POST | `/sync` | Trigger a manual sync |
| GET | `/donors` | List donor records |
| GET | `/campaigns` | List campaign records |
| GET | `/donations` | List donation records |

Full route reference: `plugins-pro/paid/donorbox/` OpenAPI spec.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/donorbox/` | Donorbox pro REST API |
| `/donorbox/webhook` | Webhook receiver |

## Examples

Trigger a manual sync:

```bash
curl -X POST -H 'Authorization: Bearer $TOKEN' \
  https://api.example.com/donorbox/sync
```

List donors:

```bash
curl -H 'Authorization: Bearer $TOKEN' \
  'https://api.example.com/donorbox/donors?limit=50'
```

Configure the Donorbox webhook URL in your Donorbox dashboard:

```
https://api.example.com/donorbox/webhook
```

Set `DONORBOX_WEBHOOK_SECRET` to the same secret configured in your Donorbox dashboard to enable signature verification.

## Source

Source-available (license required to run): [`plugins-pro/paid/donorbox/`](https://github.com/nself-org/plugins-pro/tree/main/paid/donorbox)

Note: `plugins-pro` is a private repository. Source access is granted to ɳSelf+ subscribers and Enterprise customers.

## See Also

- [[plugin-donorbox]] — free donorbox plugin (port 3074, basic sync)
- [[Pricing]] — tier comparison
- [[Plugins]] — full plugin index

← [[Plugins]] | [[Home]] →
