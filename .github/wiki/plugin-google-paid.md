# Google Plugin (Paid)

> Google OAuth2 token management with proxy APIs for Gmail, Drive, Calendar, and Sheets. Manages token refresh automatically. **Pro plugin in the ɳClaw bundle.**

> **Requires:** ɳClaw bundle or ɳSelf+ license. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install google
```

## What It Does

Provides Google OAuth2 token management and proxy APIs for Gmail, Drive, Calendar, and Sheets. Handles automatic token refresh so your application never needs to manage expiry. Exposes unified REST endpoints for all four Google services under a single authenticated proxy.

This is the paid variant running on port 3716. It is separate from the core system Google plugin (port 9003).

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `GOOGLE_PORT` | `3716` | Port the Google plugin service listens on |
| `GOOGLE_CLIENT_ID` | — | Google OAuth2 client ID |
| `GOOGLE_OAUTH_CLIENT_SECRET` | — | Google OAuth2 client secret |
| `GOOGLE_REDIRECT_URI` | `https://nself.org/oauth/google/callback` | OAuth2 redirect URI registered in Google Cloud Console |
| `GOOGLE_INTERNAL_SECRET` | — | Internal auth secret (falls back to `PLUGIN_INTERNAL_SECRET`) |

## Ports

| Port | Purpose |
|------|---------|
| `3716` | Google paid plugin HTTP service |

## Database Tables

Tables added to your Postgres database (multi-app isolated via `source_account_id`):

- `np_google_accounts` — Google account records per user
- `np_google_tokens` — OAuth2 access and refresh tokens per user
- `np_google_sync_log` — Sync event log for Google API operations

## Nginx Routes

| Route | Description |
|-------|-------------|
| `/google/` | Proxied to Google plugin service on port 3716 |

## Bundle

This plugin is part of the **ɳClaw bundle** ($0.99/mo or $9.99/yr). Also included with ɳSelf+ ($3.99/mo or $39.99/yr).

## Docker Image

```bash
docker pull nself/nself-google-paid:latest
```

## Related Pages

- [[plugin-google]] — Base Google plugin reference
- [[bundle-claw]] — ɳClaw bundle overview
- [[Home]]
