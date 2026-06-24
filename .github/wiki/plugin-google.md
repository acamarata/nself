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
| `GOOGLE_PORT` | `3716` | Port the Google plugin service listens on |
| `GOOGLE_CLIENT_ID` | — | Google OAuth2 client ID |
| `GOOGLE_CLIENT_SECRET` | — | Google OAuth2 client secret |
| `GOOGLE_REDIRECT_URI` | — | OAuth2 redirect URI registered in Google Cloud Console |

## Ports

| Port | Purpose |
|------|---------|
| `3716` | Google plugin HTTP service |

## Database Tables

4 tables added to your Postgres database.

- `np_google_accounts`, connected Google account registry per user
- `np_google_tokens`, OAuth2 access and refresh tokens (service-only, never exposed via GraphQL)
- `np_gmail_dlq`, Gmail dead-letter queue for messages that failed after retry exhaustion
- `np_google_audit_log`, one row per handler invocation for audit trails

### Row-level security

Every `np_google_*` table enforces the Multi-App Isolation Convention Wall. Each table
carries `source_account_id TEXT NOT NULL DEFAULT 'primary'`, and the Hasura row filter
`{"source_account_id": {"_eq": "X-Hasura-Source-Account-Id"}}` is applied on every role.
This keeps independent apps inside one nSelf deploy from reading each other's rows.
`np_google_tokens` holds raw OAuth credentials, so it grants no GraphQL permissions to any
role: only the plugin service (admin secret) reads or writes it. Metadata lives in
`hasura/metadata/databases/default/tables/`.

## Distribution

This is a Go plugin. It ships as a signed tarball through `ping.nself.org` and is fetched
by `nself plugin install google` after license validation. It does not publish a Docker Hub
image; the Docker image pipeline targets Rust plugins only, and the publish workflow skips
non-Rust plugins by design.

## Nginx Routes

| Route | Description |
|-------|-------------|
| `/google/` | Proxied to Google plugin service on port 3716 |
