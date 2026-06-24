# Email Plugin

> Core transactional email delivery via Elastic Email. Logs every message to Postgres with per-account sender configuration and delivery status tracking. **Pro plugin — requires license.**

## Tier required

| Tier | Includes this plugin? |
|------|-----------------------|
| Free | No |
| Any bundle ($0.99/mo) | Yes |
| ɳSelf+ ($3.99/mo) | Yes |

**Minimum tier:** Any paid bundle or ɳSelf+.

## Bundle membership

This plugin is a standalone utility — it is not exclusive to any bundle. Any active subscription ($0.99/mo or higher) unlocks it.

## Install

```bash
# Set your license key first
nself license set YOUR_LICENSE_KEY

# Install the email plugin
nself plugin install email
```

After install, set your Elastic Email API key:

```bash
nself env set EMAIL_ELASTIC_API_KEY=your_elastic_email_api_key
nself env set EMAIL_FROM_ADDRESS=noreply@yourdomain.com
nself env set EMAIL_FROM_NAME="Your App"
nself restart email
```

## Description

The email plugin provides a dedicated HTTP service (port 9008) for transactional email delivery via the Elastic Email v4 API. It is distinct from the built-in SMTP stub and from the `notify` plugin — it is purpose-built for per-account email routing with a full message log.

Every outbound email is recorded in Postgres with delivery status tracking. You can query message history, check per-account delivery statistics, and update sender configuration without restarting the service.

SSRF is not a concern because the plugin only calls the hardcoded Elastic Email API endpoint. No user-supplied URLs are accepted.

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | Postgres connection string |
| `EMAIL_ELASTIC_API_KEY` | Yes | — | Elastic Email v4 API key |
| `EMAIL_FROM_ADDRESS` | No | — | Default sender email address |
| `EMAIL_FROM_NAME` | No | `nSelf` | Default sender display name |
| `EMAIL_PORT` | No | `9008` | HTTP server port |
| `EMAIL_HOST` | No | `0.0.0.0` | HTTP bind address |
| `EMAIL_INTERNAL_SECRET` | No | — | Bearer token for API authentication |
| `EMAIL_RATE_LIMIT_RPM` | No | `60` | Max emails per minute |
| `EMAIL_RETENTION_DAYS` | No | `30` | Message log retention in days |

### Obtaining an Elastic Email API key

1. Sign up at [elasticemail.com](https://elasticemail.com).
2. Go to Settings > API.
3. Create a key with Send permission.
4. Set it as `EMAIL_ELASTIC_API_KEY` in your nSelf environment.

## Ports

| Service | Port |
|---------|------|
| plugin-email HTTP API | 9008 |

## Database schema

The plugin creates two tables with Multi-App Isolation enforced via `source_account_id`:

| Table | Purpose |
|---|---|
| `np_email_configs` | Per-account sender configuration (from address, provider, rate limit) |
| `np_email_messages` | Outbound message log with delivery status |

All rows carry `source_account_id TEXT NOT NULL DEFAULT 'primary'`. Hasura row filters enforce `{"source_account_id":{"_eq":"X-Hasura-Source-Account-Id"}}` on all user-role permissions.

## REST API

All `/email/*` endpoints require `Authorization: Bearer <EMAIL_INTERNAL_SECRET>` when `EMAIL_INTERNAL_SECRET` is set. `/health` and `/ready` are unauthenticated.

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Returns `{"status":"ok","plugin":"email"}` |
| GET | `/ready` | Readiness check — pings the database |
| POST | `/email/send` | Send a transactional email |
| GET | `/email/messages` | List recent outbound messages (last 50) |
| GET | `/email/messages/{id}` | Get a message by ID |
| GET | `/email/config` | Get sender configuration for this account |
| PUT | `/email/config` | Upsert sender configuration |
| GET | `/email/stats` | Delivery statistics by status |

### POST /email/send

Request body:

```json
{
  "to": "recipient@example.com",
  "subject": "Your subject line",
  "html": "<p>HTML body</p>",
  "text": "Plain text body"
}
```

Response (200):

```json
{
  "status": "sent",
  "message_id": "elastic-1234567890"
}
```

### PUT /email/config

```json
{
  "from_address": "noreply@yourdomain.com",
  "from_name": "Your App",
  "provider": "elastic_email",
  "api_key": "your-api-key",
  "rate_limit_rpm": 60,
  "enabled": true
}
```

## Examples

### Send a password reset email

```bash
curl -X POST http://localhost:9008/email/send \
  -H "Authorization: Bearer $EMAIL_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "subject": "Reset your password",
    "html": "<p>Click <a href=\"https://app.example.com/reset?token=abc\">here</a> to reset.</p>",
    "text": "Reset link: https://app.example.com/reset?token=abc"
  }'
```

### Check delivery stats

```bash
curl -H "Authorization: Bearer $EMAIL_INTERNAL_SECRET" \
  http://localhost:9008/email/stats
# {"sent":142,"failed":3,"queued":0}
```

### List recent messages

```bash
curl -H "Authorization: Bearer $EMAIL_INTERNAL_SECRET" \
  http://localhost:9008/email/messages
```

### Configure sender via API

```bash
curl -X PUT http://localhost:9008/email/config \
  -H "Authorization: Bearer $EMAIL_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "from_address": "hello@yourdomain.com",
    "from_name": "Your Product",
    "api_key": "EL-abc123",
    "rate_limit_rpm": 120
  }'
```

## Docker

```bash
docker pull nself/plugin-email:latest

docker run -p 9008:9008 \
  -e DATABASE_URL="postgres://..." \
  -e EMAIL_ELASTIC_API_KEY="your-key" \
  -e EMAIL_FROM_ADDRESS="noreply@example.com" \
  nself/plugin-email:latest
```

## Health check

```bash
curl http://localhost:9008/health
# {"status":"ok","plugin":"email"}

curl http://localhost:9008/ready
# {"status":"ready"}
```

## License gate

This plugin validates your nSelf license on install and startup. The license check contacts `ping.nself.org/license/validate`. The plugin fails to start if the license is invalid or expired.

To skip validation in offline development environments:

```bash
export NSELF_LICENSE_SKIP_VERIFY=1
```

## Source code

Source is available at `plugins-pro/paid/email/` in the private `nself-org/plugins-pro` repository. Access requires an active nSelf subscription.

## See also

- [[plugin-notify]] — multi-channel notifications (email, SMS, push, Slack, Telegram)
- [[Feature-Email]] — built-in SMTP configuration and Mailpit dev capture
- [[cmd-plugin]] — plugin management commands
- [[Plugins]] — full plugin catalog

---

← [[Plugins]] | [[Home]]
