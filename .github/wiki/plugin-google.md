# plugin-google

> Google APIs proxy. Manages OAuth2 tokens with AES-256-GCM encryption and automatic refresh.
> Exposes unified endpoints for Gmail, Drive, Calendar, Sheets, Contacts, GCP, and Gemini probes.
> **Pro plugin (ɳClaw bundle).**

**Requires:** ɳClaw bundle or ɳSelf+ subscription.

```bash
nself license set <your-license-key>
nself plugin install google
```

## What It Does

Provides OAuth2 token management and proxy REST APIs for all major Google services. Handles
automatic token refresh so your application never manages expiry. All tokens are stored
encrypted with AES-256-GCM. The service exposes Gmail, Drive, Calendar, Sheets, Contacts,
and GCP endpoints under a single authenticated proxy on port 9003.

## OAuth Setup

1. Create a project at [console.cloud.google.com](https://console.cloud.google.com).
2. Enable APIs: Gmail API, Drive API, Calendar API, Sheets API, People API.
3. Create OAuth 2.0 credentials (Web application type).
4. Set the authorised redirect URI to `https://<your-domain>/google/oauth/callback`.
5. Copy the Client ID and Client Secret into the env vars below.

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GOOGLE_CLIENT_ID` | Yes | OAuth2 client ID from Google Console |
| `GOOGLE_CLIENT_SECRET` | Yes | OAuth2 client secret |
| `GOOGLE_REDIRECT_URI` | Yes | Authorised redirect URI (must match Console exactly) |
| `GOOGLE_PLUGIN_PORT` | No | HTTP server port (default: `9003`) |
| `GOOGLE_ENCRYPTION_KEY` | Yes | AES-256-GCM key, 32 bytes hex-encoded, for token encryption |
| `NSELF_DB_URL` | Yes | PostgreSQL connection string |
| `NSELF_LICENSE_KEY` | Yes | License key (validated against `ping.nself.org`) |

## API Endpoints

### OAuth Flow

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/google/oauth/start` | Initiate OAuth2 authorisation flow |
| `GET` | `/google/oauth/callback` | OAuth2 callback (set as redirect URI in Console) |
| `POST` | `/google/oauth/revoke` | Revoke tokens and remove the connected account |
| `GET` | `/google/accounts` | List all connected Google accounts |
| `GET` | `/google/accounts/:id` | Get a specific account |
| `DELETE` | `/google/accounts/:id` | Disconnect account and delete tokens |

### Gmail

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/google/gmail/messages` | List messages (params: `q`, `maxResults`, `pageToken`) |
| `GET` | `/google/gmail/messages/:id` | Get a single message with body and headers |
| `POST` | `/google/gmail/send` | Send an email (structured JSON or raw RFC 2822) |
| `GET` | `/google/gmail/labels` | List all labels |
| `POST` | `/google/gmail/watch` | Subscribe to Gmail push notifications |
| `DELETE` | `/google/gmail/watch` | Unsubscribe from push notifications |

### Google Drive

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/google/drive/files` | List files (params: `q`, `fields`, `pageToken`) |
| `GET` | `/google/drive/files/:id` | Get file metadata |
| `GET` | `/google/drive/files/:id/content` | Download file content |
| `POST` | `/google/drive/files` | Upload a file (multipart) |
| `DELETE` | `/google/drive/files/:id` | Move file to trash |
| `POST` | `/google/drive/rag` | Index a Drive file for RAG retrieval |

### Google Calendar

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/google/calendar/events` | List events (params: `timeMin`, `timeMax`, `calendarId`) |
| `GET` | `/google/calendar/events/:id` | Get a single event |
| `POST` | `/google/calendar/events` | Create a calendar event |
| `PUT` | `/google/calendar/events/:id` | Update a calendar event |
| `DELETE` | `/google/calendar/events/:id` | Delete a calendar event |
| `GET` | `/google/calendar/calendars` | List all calendars for the connected account |

### Google Sheets

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/google/sheets/:id` | Get spreadsheet metadata |
| `GET` | `/google/sheets/:id/values/:range` | Read cell values (A1 notation) |
| `PUT` | `/google/sheets/:id/values/:range` | Write cell values |
| `POST` | `/google/sheets/:id/batchUpdate` | Batch spreadsheet operations |

### Contacts

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/google/contacts` | List contacts |
| `GET` | `/google/contacts/:resourceName` | Get a contact by resource name |

### Internal

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check (readiness probe) |
| `POST` | `/google/internal/token` | Internal token exchange (service-to-service) |
| `GET` | `/google/internal/tools` | Tool listing for ɳClaw AI integration |
| `POST` | `/google/gcp/probe` | GCP project metadata probe |
| `POST` | `/google/gemini/probe` | Gemini AI availability probe |

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_google_accounts` | Connected Google accounts per user (`source_account_id` isolated) |
| `np_google_tokens` | AES-256-GCM encrypted OAuth tokens (linked via FK to accounts) |
| `np_google_audit_log` | API call audit trail: surface, action, status, timestamp |

All `np_google_accounts` and `np_google_audit_log` rows carry `source_account_id` for
multi-app isolation per the nSelf Multi-Tenant Convention Wall (Convention A).

## Hasura Row-Level Security

Hasura metadata applies `source_account_id = X-Hasura-Source-Account-Id` row filters on
`np_google_accounts` and `np_google_audit_log`. Token rows are isolated transitively via
the `account_id` FK on `np_google_tokens`.

## SSRF Guard

All outbound HTTP requests target `googleapis.com` only. The redirect URI is validated
server-side against `GOOGLE_REDIRECT_URI`. User-supplied URLs are never followed as
outbound targets. There are no user-overridable outbound URL fields.

## Nginx Routes

```nginx
location /google/ {
    proxy_pass http://plugin-google:9003/google/;
}
```

## Port

| Port | Protocol | Purpose |
|------|----------|---------|
| `9003` | HTTP | plugin-google REST API |

---

[[Home]] | [[Plugin-Overview]] | [[bundle-nclaw]]
