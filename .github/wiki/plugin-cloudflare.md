# Cloudflare Plugin

> Full Cloudflare zone management: DNS records, R2 object storage, cache purging, analytics, and webhook sync. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|----------------------|
| Free | $0 | $0 | No |
| Basic | $0.99/mo | $9.99/yr | Yes |
| Pro | $1.99/mo | $19.99/yr | Yes |
| Elite | $4.99/mo | $49.99/yr | Yes |
| Business | $9.99/mo | $99.99/yr | Yes |
| Business+ | $49.99/mo | $499.99/yr | Yes |
| Enterprise | $99.99/mo | $999.99/yr | Yes |

**Minimum tier:** Basic (this is a `tier: pro` plugin per F07-PRICING-TIERS).

## Bundle membership

Not currently bundled in a named product bundle. Access requires a Basic tier subscription or higher.

Or get all bundles and all apps via **ɳSelf+** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install cloudflare
nself build
```

The license is validated against `ping.nself.org/license/validate`. Tier is checked server-side; insufficient tier returns an error.

## Description

The Cloudflare plugin integrates with the Cloudflare API to provide full zone management from your nSelf stack. It handles DNS record CRUD (A, AAAA, CNAME, MX, TXT), R2 object storage bucket operations, cache purge requests by URL or path, and periodic analytics snapshots. All zone and record changes are written to Postgres for auditability and replayed on reconnect.

The plugin holds your `CF_API_TOKEN` server-side only. Client apps interact through the local REST API at port 3024; the token never leaves the server. Zone data, DNS records, and analytics are cached locally to reduce Cloudflare API calls.

**This plugin is distinct from the `cdn` plugin (port 3036).** The cdn plugin handles multi-provider cache purging (Cloudflare, Bunny, Fastly) and HMAC-SHA256 signed URL generation. The cloudflare plugin handles full Cloudflare-specific zone management: DNS, R2, WAF rules, and account-level analytics. If you only need cache purge and signed URLs across multiple CDN providers, use the cdn plugin. If your infrastructure runs on Cloudflare and you need DNS, R2, and zone control, use this plugin.

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `CF_API_TOKEN` | Yes | — | Cloudflare API token with zone read/edit and account permissions |
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `CF_ACCOUNT_ID` | No | — | Cloudflare account ID (required for R2 bucket operations) |
| `CF_ZONE_IDS` | No | — | Comma-separated zone IDs to scope sync |
| `CF_R2_ACCESS_KEY` | No | — | R2 access key ID |
| `CF_R2_SECRET_KEY` | No | — | R2 secret access key |
| `CF_PLUGIN_PORT` | No | `3024` | Port override for the plugin service |
| `CF_PLUGIN_HOST` | No | `127.0.0.1` | Host binding override |
| `CF_LOG_LEVEL` | No | `info` | Log verbosity |
| `CF_APP_IDS` | No | — | Application IDs for multi-app isolation scope |
| `CF_SYNC_INTERVAL` | No | `3600` | Sync interval in seconds |

Reference vault credentials. Never hardcode secrets.

## Ports

| Port | Purpose |
|------|---------|
| `3024` | Cloudflare plugin HTTP service |

Bound to `127.0.0.1` per nSelf service-binding rules. Reach via Nginx, never directly.

## Database Tables

| Table | Description |
|-------|-------------|
| `np_cf_zones` | Synced Cloudflare zone records |
| `np_cf_dns_records` | DNS A/AAAA/CNAME/MX/TXT records |
| `np_cf_r2_buckets` | R2 bucket metadata |
| `np_cf_cache_purge_log` | Cache purge history |
| `np_cf_analytics` | Zone analytics snapshots |
| `np_cf_webhook_events` | Incoming Cloudflare webhook events |

All tables use `source_account_id` for multi-app isolation.

## REST API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness probe |
| GET | `/ready` | Readiness probe |
| GET | `/api/v1/zones` | List synced zones |
| POST | `/api/v1/zones` | Add a zone |
| GET | `/api/v1/zones/{id}` | Get zone detail |
| DELETE | `/api/v1/zones/{id}` | Remove a zone |
| GET | `/api/v1/dns-records` | List DNS records |
| POST | `/api/v1/dns-records` | Create a DNS record |
| PUT | `/api/v1/dns-records/{id}` | Update a DNS record |
| DELETE | `/api/v1/dns-records/{id}` | Delete a DNS record |
| POST | `/api/v1/cache/purge` | Purge cache for URLs or paths |
| GET | `/api/v1/cache/purge-log` | View purge history |
| GET | `/api/v1/r2/buckets` | List R2 buckets |
| POST | `/api/v1/r2/buckets` | Create an R2 bucket |
| DELETE | `/api/v1/r2/buckets/{id}` | Delete an R2 bucket |
| GET | `/api/v1/analytics` | Retrieve zone analytics snapshots |
| GET | `/api/v1/stats` | Plugin statistics |

All endpoints require a bearer token.

## Nginx Routes

| Route | Description |
|-------|-------------|
| `/cloudflare/` | Proxied to Cloudflare plugin service on port 3024 |

## Examples

List zones:

```bash
curl -H 'Authorization: Bearer $TOKEN' https://api.example.com/cloudflare/api/v1/zones
```

Create a DNS A record:

```bash
curl -X POST -H 'Authorization: Bearer $TOKEN' \
  -H 'Content-Type: application/json' \
  https://api.example.com/cloudflare/api/v1/dns-records \
  -d '{"zone_id":"zone_xxx","type":"A","name":"app","content":"1.2.3.4","ttl":300}'
```

Purge cache for a path:

```bash
curl -X POST -H 'Authorization: Bearer $TOKEN' \
  -H 'Content-Type: application/json' \
  https://api.example.com/cloudflare/api/v1/cache/purge \
  -d '{"zone_id":"zone_xxx","files":["https://example.com/style.css"]}'
```

List R2 buckets:

```bash
curl -H 'Authorization: Bearer $TOKEN' https://api.example.com/cloudflare/api/v1/r2/buckets
```

## Source

Source-available (license required to run): [`plugins-pro/paid/cloudflare/`](https://github.com/nself-org/plugins-pro/tree/main/paid/cloudflare)

Note: `plugins-pro` is a private repository. Source access is granted to ɳSelf+ subscribers and Enterprise customers.

## See Also

- [[plugin-cdn]] — multi-provider cache purge and signed URL generation (port 3036); use instead of this plugin when targeting multiple CDN providers
- [[plugin-ddns]] — dynamic DNS updates for auto-rotating IPs
- [[plugin-object-storage]] — provider-agnostic object storage
- [[Pricing]] — tier comparison
- [[Plugins]] — full plugin index

← [[Plugins]] | [[Home]] →
