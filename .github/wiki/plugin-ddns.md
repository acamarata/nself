# DDNS Plugin

> Dynamic DNS service that monitors your server's public IP and updates DNS records when it changes. **Pro plugin** (port 3217, unbundled).

> **Requires:** Pro license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install ddns
```

## What It Does

The ddns plugin runs as a background HTTP service on port 3217. It detects your server's current public IP address (via `api.ipify.org` with a fallback to `api.my-ip.io`) and, for each DNS record you configure, updates the record at your DNS provider when the detected IP differs from the last known value. Each detection and update attempt is persisted to Postgres so you can audit IP history and debug failed updates.

Records are configured at runtime through the plugin's REST API and stored per record in the database, not through environment variables. This lets you manage multiple domains and providers from a single running instance.

## Supported Providers

| Provider | Status | Notes |
|----------|--------|-------|
| Cloudflare | Implemented | Updates an A record via the Cloudflare API using a per-record zone ID, record ID, and API token. |
| DuckDNS, No-IP, Dynu | Configurable | Stored as provider config rows; useful for IP tracking and update logging. |
| Route53 | Planned | Listed in plugin metadata; not yet wired into the update path. |

Cloudflare is the actively wired update target. Provider, domain, token, and zone details are supplied per record through the API, never hardcoded.

## Configuration (Environment Variables)

The plugin reads only the following environment variables. Per-record DNS credentials are supplied through the API, not the environment.

| Env Var | Default | Description |
|---------|---------|-------------|
| `DDNS_PLUGIN_PORT` | `3217` | Port the DDNS service listens on. |
| `DDNS_HOST` | `0.0.0.0` | Bind address for the service. |
| `DATABASE_URL` | auto-provided by nself | PostgreSQL connection string. |
| `DDNS_API_KEY` | (none) | Optional. When set, requests to `/api/v1/*` must send a matching `X-API-Key` header. |

`DATABASE_URL` is supplied automatically by the nSelf backend. Set `DDNS_API_KEY` to require an API key for all data endpoints.

## API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/health` | Liveness probe. |
| GET | `/ready` | Readiness probe (checks the database). |
| GET | `/api/v1/ip` | Return the currently detected public IP. |
| GET / POST | `/api/v1/configs` | List or create DNS record configurations. |
| GET / PUT / DELETE | `/api/v1/configs/{id}` | Read, update, or delete a configuration. |
| POST | `/api/v1/configs/{id}/update` | Force an update for one record. |
| POST | `/api/v1/update` | Run an update pass across all enabled records. |
| GET | `/api/v1/update-log` | List recent IP change and update events. |
| GET | `/api/v1/stats` | Summary statistics for configured records. |

## Polling and Scheduling

Each configuration carries its own `check_interval` (seconds, default 300). The service does not expose any public route; it is a background worker plus an internal management API. To trigger update passes on a schedule from outside the container, call `POST /api/v1/update` from a systemd timer or cron job, for example:

```bash
# /etc/cron.d/ddns: run an update pass every 5 minutes
*/5 * * * * root curl -fsS -X POST -H "X-API-Key: $DDNS_API_KEY" http://localhost:3217/api/v1/update >/dev/null 2>&1
```

A systemd timer can wrap the same `curl` call in a `.service` unit triggered by an `OnUnitActiveSec=5min` timer.

## Database Tables

Two tables are added to your Postgres database. Both carry `source_account_id` for multi-app isolation.

- `np_ddns_config`, configured DNS records and their last known IP values.
- `np_ddns_update_log`, history of IP change detections and DNS update results.

## Ports

| Port | Purpose |
|------|---------|
| `3217` | DDNS management HTTP service (background service; no public nginx routes). |

## Nginx Routes

None. This is a background service only and does not expose any public HTTP routes.

---

Related: [[Plugin-Overview]] · [[plugin-cloudflare]] · [[Home]]
