# Cloudflare Plugin

> Cloudflare integration for DNS zone management, R2 object storage, cache purging, analytics retrieval, and webhook event sync. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install cloudflare
```

## What It Does

Integrates with the Cloudflare API to manage DNS zones and records, interact with R2 object storage, purge cache, retrieve analytics, and sync webhook events. Provides a local REST API that wraps Cloudflare's API with persistent state stored in Postgres. All zone and record changes are logged for auditability.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CLOUDFLARE_PORT` | `3024` | Port the Cloudflare plugin service listens on |
| `CLOUDFLARE_API_TOKEN` | — | Cloudflare API token with zone and account permissions |
| `CLOUDFLARE_ZONE_ID` | — | Default Cloudflare zone ID |
| `CLOUDFLARE_ACCOUNT_ID` | — | Cloudflare account ID (required for R2) |

## Ports

| Port | Purpose |
|------|---------|
| `3024` | Cloudflare plugin HTTP service |

## Database Tables

6 tables added to your Postgres database.

- `np_cloudflare_zones` — Synced zone records
- `np_cloudflare_dns_records` — DNS A/AAAA/CNAME/MX/TXT records
- `np_cloudflare_r2_buckets` — R2 bucket metadata
- `np_cloudflare_cache_rules` — Cache purge rules and history
- `np_cloudflare_analytics` — Retrieved analytics snapshots
- `np_cloudflare_webhooks` — Incoming Cloudflare webhook events

## Nginx Routes

| Route | Description |
|-------|-------------|
| `/cloudflare/` | Proxied to Cloudflare plugin service on port 3024 |
