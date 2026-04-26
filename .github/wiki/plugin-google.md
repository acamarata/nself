# Google Plugin

> Google OAuth2 token management with proxy APIs for Gmail, Drive, Calendar, and Sheets. Manages token refresh automatically. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install google
```

## What It Does

Provides Google OAuth2 token management and proxy APIs for Gmail, Drive, Calendar, and Sheets. Handles automatic token refresh so your application never needs to manage expiry. Exposes unified REST endpoints for all four Google services under a single authenticated proxy.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `GOOGLE_PORT` | `3717` | Port the Google plugin service listens on |
| `GOOGLE_CLIENT_ID` | — | Google OAuth2 client ID |
| `GOOGLE_CLIENT_SECRET` | — | Google OAuth2 client secret |
| `GOOGLE_REDIRECT_URI` | — | OAuth2 redirect URI registered in Google Cloud Console |

## Ports

| Port | Purpose |
|------|---------|
| `3717` | Google plugin HTTP service |

## Database Tables

2 tables added to your Postgres database.

- `np_google_tokens`, OAuth2 access and refresh tokens per user
- `np_google_sync_log`, Sync event log for Google API operations

## Nginx Routes

| Route | Description |
|-------|-------------|
| `/google/` | Proxied to Google plugin service on port 3717 |
