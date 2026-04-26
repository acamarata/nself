# DDNS Plugin

> Dynamic DNS with Cloudflare IP monitoring, automatically updates DNS records when your server's public IP changes. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install ddns
```

## What It Does

Monitors your server's public IPv4 and IPv6 addresses on a configurable interval and automatically updates Cloudflare DNS A and AAAA records when a change is detected. Operates in graceful degraded mode if the Cloudflare API is unreachable, retrying on the next interval rather than crashing. All IP changes and update attempts are persisted to the database for audit and debugging.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DDNS_PORT` | `3217` | Port the DDNS plugin status service listens on |
| `DDNS_CLOUDFLARE_TOKEN` | — | Cloudflare API token with DNS edit permissions |
| `DDNS_ZONE_ID` | — | Cloudflare zone ID for the target domain |
| `DDNS_RECORD_NAME` | — | DNS record name to keep updated (e.g. `home.example.com`) |
| `DDNS_CHECK_INTERVAL` | `5m` | Interval between public IP checks |

## Ports

| Port | Purpose |
|------|---------|
| `3217` | DDNS plugin status HTTP service (background service; no public routes) |

## Database Tables

2 tables added to your Postgres database.

- `np_ddns_records`, Tracked DNS records and last known IP values
- `np_ddns_update_log`, History of IP change detections and DNS update results

## Nginx Routes

None. This is a background service only and does not expose any public HTTP routes.
