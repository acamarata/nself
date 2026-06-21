# Google Plugin

> Google OAuth2 token management with proxy APIs for Gmail, Drive, Calendar, Sheets, Contacts, and more. Handles automatic token refresh and multi-user account management. **Pro plugin (ɳClaw bundle).**

> **Requires:** Pro license tier or ɳSelf+ subscription. Set license: `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install google
nself plugin health google
```

## What It Does

The ɳSelf Google plugin provides OAuth2 token management and unified REST proxy for Google Workspace services. Instead of managing Google tokens in your application code, the plugin handles:

- **Automatic token refresh** — tokens refresh seamlessly before expiry
- **Multi-user accounts** — store multiple Google accounts per ɳSelf user
- **Encrypted token storage** — AES-256-GCM encryption in PostgreSQL
- **Unified REST API** — single proxy for Gmail, Calendar, Drive, Sheets, Contacts
- **OAuth2 setup wizard** — built-in `POST /oauth/authorize` flow

Exposed endpoints cover full Workspace access: Gmail label management, inbox search, send emails, Calendar create/modify events, Drive file upload/sync/RAG indexing, Sheets read/write, Contacts search.

## Getting Started

### 1. Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project (or use existing)
3. Enable these APIs: Gmail API, Google Calendar API, Google Drive API, Google Sheets API, Google Contacts API
4. Create an OAuth2 credential (Web application type)
5. Add authorized redirect URI: `http://localhost:3714/auth/callback` (dev) or your production domain

### 2. Configure Environment

```bash
export GOOGLE_CLIENT_ID="<your-client-id>"
export GOOGLE_CLIENT_SECRET="<your-client-secret>"
export GOOGLE_REDIRECT_URI="<your-domain>/auth/callback"
```

### 3. Test OAuth Flow

```bash
curl -X POST http://localhost:3714/google/oauth/authorize \
  -H "Authorization: Bearer $NSELF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@gmail.com"}'
```

Complete the redirect URL in a browser. Your token is now stored encrypted.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `GOOGLE_PORT` | `3716` | Port the Google plugin listens on |
| `GOOGLE_HOST` | `localhost` | Host binding |
| `GOOGLE_CLIENT_ID` | — | OAuth2 client ID from Google Cloud |
| `GOOGLE_CLIENT_SECRET` | — | OAuth2 client secret (never log) |
| `GOOGLE_REDIRECT_URI` | — | Callback URI registered in Google Cloud Console |
| `GOOGLE_INTERNAL_SECRET` | — | Internal plugin authentication key |

## Ports

| Port | Purpose |
|------|---------|
| `3716` | Google plugin HTTP service |

## Database

The plugin creates 2 tables:

- **`np_google_accounts`** — User accounts linked to Google (email, OAuth2 user ID, active status, scopes granted)
- **`np_google_tokens`** — Encrypted access/refresh tokens per account (AES-256-GCM)

Tokens are never exposed via API — only account info and token status are queryable.

## Nginx Routes

All requests to `/google/*` are proxied to the plugin on port 3716:

```nginx
location /google/ {
  proxy_pass http://localhost:3716/;
}
```

User must present valid `Authorization: Bearer` token.

## API Endpoints

Partial list — full spec in `plugin.json`:

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/oauth/authorize` | Start OAuth2 flow |
| GET | `/accounts` | List linked accounts |
| GET | `/gmail/messages` | Search inbox |
| POST | `/gmail/send` | Send email |
| GET | `/calendar/calendars` | List calendars |
| POST | `/calendar/events` | Create event |
| GET | `/drive/files` | List Drive files |
| POST | `/drive/upload` | Upload file |
| POST | `/sheets/read` | Read sheet rows |
| GET | `/contacts` | Search contacts |

## Licensing

The Google plugin requires an active Pro license. License validation happens at plugin install time and on every startup. If your license expires, the plugin remains readable but cannot create new authorizations.

Check license status:

```bash
nself license status
nself plugin health google
```

## Support

For issues: [ɳSelf Discord](https://discord.gg/nself) · Docs: [ɳSelf Wiki](https://github.com/nself-org/cli/wiki)
