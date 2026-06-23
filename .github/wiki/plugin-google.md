# ɳGoogle Plugin

OAuth2-based Google APIs integration for ɳClaw and ɳSelf+. Manage Gmail, Calendar, Drive, Sheets, and more with automatic token refresh and secure token storage.

## Features

- **OAuth2 Token Management** — Automatic refresh, secure storage in PostgreSQL with tenant isolation
- **Gmail** — Read messages, search, send, manage labels, batch operations
- **Google Calendar** — Create/update/delete events, manage calendars, check availability, out-of-office status
- **Google Drive** — List files, download, upload, RAG indexing and sync
- **Google Contacts** — List and search contacts
- **Google Sheets** — Read/write/append cells, clear ranges, batch operations
- **Gemini AI** — Integrate Gemini API for AI-powered features
- **GCP Provisioning** — Provision GCP service accounts and OAuth credentials on demand
- **Multi-Account** — Switch between multiple Google accounts per user

## Installation

### Requirements

- ɳSelf v1.0.0+
- ɳClaw Bundle or ɳSelf+ license
- Google OAuth2 credentials (Client ID + Client Secret)

### Install the Plugin

```bash
nself plugin install google
```

The CLI will prompt for:
- `GOOGLE_CLIENT_ID` — OAuth app client ID
- `GOOGLE_OAUTH_CLIENT_SECRET` — OAuth app client secret
- `GOOGLE_REDIRECT_URI` (optional) — defaults to `http://localhost:3714/auth/callback`

### Verify Installation

```bash
curl http://localhost:3716/health -H "Authorization: Bearer $(nself auth token)"
# Output: {"status":"ok","version":"1.1.2"}
```

## Configuration

The plugin runs on port **3716** (configurable via `GOOGLE_PORT` env var).

### Environment Variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `GOOGLE_CLIENT_ID` | Yes | — | OAuth2 app client ID from Google Console |
| `GOOGLE_OAUTH_CLIENT_SECRET` | Yes | — | OAuth2 app client secret |
| `GOOGLE_REDIRECT_URI` | No | `http://localhost:3714/auth/callback` | OAuth2 redirect URI |
| `GOOGLE_PORT` | No | 3716 | Port for the plugin service |
| `GOOGLE_HOST` | No | 0.0.0.0 | Host binding |
| `GOOGLE_INTERNAL_SECRET` | No | auto-generated | Internal request signing key |
| `DATABASE_URL` | Yes | inherited | PostgreSQL connection string |

### Database Tables

- **`np_google_accounts`** — Stores OAuth account metadata (email, user_id, scopes, active status)
- **`np_google_tokens`** — Stores OAuth tokens (access_token, refresh_token, expires_at)

Both tables use **`source_account_id`** for multi-app isolation. Tenant isolation is enforced by Hasura row-level security (RLS).

## Usage Examples

### Initialize OAuth Account

```bash
curl -X POST http://localhost:3716/accounts \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@gmail.com"}'
```

### List Connected Accounts

```bash
curl http://localhost:3716/accounts \
  -H "Authorization: Bearer $TOKEN"
```

### Send an Email

```bash
curl -X POST http://localhost:3716/gmail/send \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "recipient@example.com",
    "subject": "Hello",
    "body": "This is a test email"
  }'
```

### Get Calendar Events

```bash
curl http://localhost:3716/calendar/events \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Google-Account-ID: {accountId}"
```

### Upload to Drive and Index for RAG

```bash
curl -X POST http://localhost:3716/drive/rag/index \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"file_id":"..."}'
```

### Read/Write Sheets

```bash
# Read a range
curl -X POST http://localhost:3716/sheets/read \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"spreadsheet_id":"...", "range":"Sheet1!A1:Z"}'

# Write a range
curl -X POST http://localhost:3716/sheets/write \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"spreadsheet_id":"...", "range":"Sheet1!A1", "values":[["Header"]]}'
```

## API Endpoints

### Accounts

- `GET /accounts` — List all connected accounts
- `POST /accounts` — Add a new Google account (OAuth)
- `GET /accounts/{id}` — Get account details
- `DELETE /accounts/{id}` — Remove account

### Gmail

- `GET /gmail/messages` — List messages (paginated)
- `GET /gmail/messages/{id}` — Get message details
- `POST /gmail/send` — Send an email
- `POST /gmail/batch-headers` — Fetch headers for multiple messages
- `POST /gmail/batch-modify` — Bulk label/archive operations
- `POST /gmail/labels` — Manage labels
- `POST /gmail/watch/renew` — Renew push notification subscription

### Google Calendar

- `GET /calendar/calendars` — List calendars
- `POST /calendar/calendars` — Create a calendar
- `DELETE /calendar/calendars/{cal_id}` — Delete a calendar
- `GET /calendar/events` — List events
- `POST /calendar/events` — Create an event
- `PATCH /calendar/events/{id}` — Update an event
- `DELETE /calendar/events/{id}` — Delete an event
- `POST /calendar/free-slots` — Check free availability
- `GET /calendar/settings` — Get calendar settings

### Google Drive

- `GET /drive/files` — List files
- `GET /drive/files/{id}` — Get file details
- `GET /drive/download/{id}` — Download a file
- `POST /drive/upload` — Upload a file
- `POST /drive/rag/index` — Index Drive content for RAG
- `POST /drive/rag/sync` — Sync Drive changes for RAG

### Google Contacts

- `GET /contacts` — List contacts
- `POST /contacts/search` — Search contacts
- `GET /contacts/{resource_name}` — Get contact details

### Google Sheets

- `POST /sheets/read` — Read cell ranges
- `POST /sheets/write` — Write cell ranges
- `POST /sheets/append` — Append rows
- `POST /sheets/clear` — Clear cell ranges
- `GET /sheets/{id}/values/{range}` — Get specific range values

### Gemini AI

- `POST /{user}/gemini/probe` — Check Gemini availability

### GCP Provisioning

- `GET /gcp/provision` — Check GCP project provisioning status
- `POST /gcp/provision` — Provision a GCP project and service account

### Health & Status

- `GET /health` — Health check
- `GET /ready` — Readiness check
- `GET /google/health` — Detailed Google API health
- `POST /tokens/status` — Check token status and refresh if needed
- `GET /stats` — Plugin statistics and quota usage

## OAuth Setup

### Create OAuth Credentials in Google Cloud Console

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable these APIs:
   - Gmail API
   - Google Calendar API
   - Google Drive API
   - Google Sheets API
   - Google Contacts API
   - Gemini API (optional)
4. Go to **Credentials** → **Create Credentials** → **OAuth 2.0 Client ID**
5. Choose **Desktop Application** (for self-hosted) or **Web Application** (for cloud)
6. Add authorized redirect URI: `http://localhost:3714/auth/callback` (dev) or `https://claw.nself.org/auth/callback` (prod)
7. Copy Client ID and Client Secret

### Scopes

The plugin requests these scopes:

- `https://www.googleapis.com/auth/gmail.readonly` — Read Gmail
- `https://www.googleapis.com/auth/gmail.send` — Send emails
- `https://www.googleapis.com/auth/gmail.modify` — Modify labels, archive
- `https://www.googleapis.com/auth/calendar` — Full Calendar access
- `https://www.googleapis.com/auth/drive.readonly` — Read Drive
- `https://www.googleapis.com/auth/drive.file` — Scoped Drive access
- `https://www.googleapis.com/auth/spreadsheets` — Sheets read/write
- `https://www.googleapis.com/auth/contacts.readonly` — Read Contacts
- `https://www.googleapis.com/auth/userinfo.email` — User email
- `https://www.googleapis.com/auth/userinfo.profile` — User profile (optional)

## Security

- **Token Storage** — Tokens encrypted at rest in PostgreSQL with AES-256
- **Automatic Refresh** — Expired tokens automatically refreshed server-side
- **Tenant Isolation** — Row-level security (RLS) enforced by Hasura; users cannot access other tenants' tokens
- **No Client-Side Tokens** — Tokens never exposed via GraphQL or REST responses
- **Scoped OAuth** — Users grant minimal required scopes
- **Rate Limiting** — Built-in rate limiting per account to prevent quota exhaustion

## Troubleshooting

### "Invalid OAuth token" error

The refresh token may have expired. Disconnect and reconnect the account:

```bash
curl -X DELETE http://localhost:3716/accounts/{id} \
  -H "Authorization: Bearer $TOKEN"
```

Then re-authenticate via the UI.

### "Quota exceeded" error

You've hit Google's daily quota for this API. Wait 24 hours or upgrade your GCP project quota in the Console.

### Plugin fails to start

Check logs:

```bash
docker logs nself_plugin_google
```

Ensure `DATABASE_URL`, `GOOGLE_CLIENT_ID`, and `GOOGLE_OAUTH_CLIENT_SECRET` are set.

### OAuth callback not received

Verify the redirect URI in Google Cloud Console matches `GOOGLE_REDIRECT_URI` in your nSelf config.

## Testing

The plugin includes 83 unit tests covering OAuth flow, token refresh, API calls, and error handling:

```bash
cd plugins-pro/paid/google
go test ./... -v
```

## License

This plugin requires an ɳClaw Bundle or ɳSelf+ license. See [nself.org/pricing](https://nself.org/pricing) for details.

## Support

For issues or feature requests, contact support@nself.org or open an issue on [GitHub](https://github.com/nself-org/plugins-pro).
