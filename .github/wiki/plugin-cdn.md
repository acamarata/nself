# CDN Plugin

> CDN management with cache purging, HMAC-SHA256 signed URL generation, and multi-provider support. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install cdn
```

## What It Does

Manages CDN configuration across Cloudflare, Bunny, and Fastly from a single API. Handles cache purge requests by tag, URL, or zone with persistent logging of all purge operations. Generates HMAC-SHA256 signed URLs for time-limited secure asset access. Analytics snapshots from supported providers are stored locally for reporting without repeated API calls.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CDN_PORT` | `3036` | Port the CDN plugin service listens on |
| `CDN_PROVIDER` | — | CDN provider: `cloudflare`, `bunny`, `fastly` |
| `CDN_API_KEY` | — | API key for the selected CDN provider |
| `CDN_BASE_URL` | — | CDN base URL for signed URL generation |
| `CDN_HMAC_SECRET` | — | Secret key for HMAC-SHA256 signed URL signing |

## Ports

| Port | Purpose |
|------|---------|
| `3036` | CDN plugin HTTP service |

## Database Tables

4 tables added to your Postgres database.

- `np_cdn_configs`, CDN provider configuration records
- `np_cdn_purge_log`, Cache purge request history
- `np_cdn_signed_urls`, Issued signed URL audit log
- `np_cdn_analytics`, Cached CDN analytics snapshots

## Nginx Routes

| Route | Description |
|-------|-------------|
| `/cdn/` | Proxied to CDN plugin service on port 3036 |
