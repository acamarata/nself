# ɳSelf Plugin: notify (Core)

**Pro Plugin** | **Port: 3712** | **Status: Stable** | **API v1.0.0**

Multi-channel notification delivery plugin for ɳClaw, ɳChat, ɳTask, ClawDE, and ɳFamily. Routes notifications through email, SMS, Telegram, Slack, web push, and HTTP webhooks with built-in retry, rate limiting, and delivery tracking.

---

## Installation

### Prerequisites
- ɳSelf v1.0.0 or later
- Pro license (ɳSelf+ or notify bundle)
- PostgreSQL 12+
- SMTP server (for email) / Telegram bot token / Twilio account (for SMS)

### Quick Install

```bash
# Requires ɳSelf+ or notify bundle
nself plugin install notify

# Verify
nself plugin status notify
curl http://localhost:3712/health
```

### Automatic Configuration

`nself plugin install notify` performs these steps:

1. ✅ Creates PostgreSQL tables (`np_notify_*` schema)
2. ✅ Wires up multi-tenant RLS (Hasura row filters on tenant_id)
3. ✅ Registers the plugin with nSelf daemon
4. ✅ Starts the health check loop
5. ✅ Configures system environment variables

Manual setup is rarely needed.

---

## Configuration

### Environment Variables

**Database (Required)**
- `DATABASE_URL` — PostgreSQL connection. Format: `postgresql://user:pass@host/dbname`

**Server (Optional)**
- `NOTIFY_PORT` — Listen port. Default: `3712`
- `NOTIFY_HOST` — Listen address. Default: `127.0.0.1` (local only)
- `NOTIFY_INTERNAL_SECRET` — Shared secret for health checks. Default: generated on install.

**Channels (Optional — enable as needed)**

| Channel | Required Env Vars |
|---------|-------------------|
| **Email (SMTP)** | `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` |
| **SMS (Twilio)** | `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_NUMBER` |
| **Push (FCM)** | `FCM_SERVER_KEY` |
| **Telegram** | `TELEGRAM_BOT_TOKEN` |
| **Slack** | `SLACK_WEBHOOK_URL` (optional; per-channel webhook stored in config) |
| **Webhooks** | None (user-defined per notification) |

**Logging (Optional)**
- `LOG_LEVEL` — `debug`, `info` (default), `warn`, `error`

### Example Configuration

```bash
# Email + Telegram setup
export DATABASE_URL="postgresql://nself:secret@db.local/nself"
export SMTP_HOST="mail.example.com"
export SMTP_PORT="587"
export SMTP_USER="notify@example.com"
export SMTP_PASSWORD="app-password"
export SMTP_FROM="nself-notify@example.com"
export TELEGRAM_BOT_TOKEN="1234567890:ABCDefghijKlmnopqrstuvwXYZ"

nself plugin install notify
```

---

## API Reference

### Send a Notification

**POST** `/send`

```json
{
  "channel": "email|telegram|sms|slack|webhook|webpush",
  "recipient": "user@example.com or user_id or chat_id",
  "title": "Alert Title",
  "body": "Alert message content",
  "template_name": "optional-template",
  "template_vars": {
    "key": "value"
  },
  "priority": "high|normal|low",
  "retry_count": 3,
  "meta": {
    "app": "nclaw"
  }
}
```

**Response (202 Accepted)**
```json
{
  "id": "uuid",
  "status": "pending",
  "channel": "email",
  "created_at": "2026-06-22T10:30:45Z"
}
```

### Send to Multiple Recipients

**POST** `/send/batch`

```json
{
  "channel": "email",
  "recipients": ["user1@example.com", "user2@example.com"],
  "title": "Batch Notification",
  "body": "Message",
  "priority": "normal"
}
```

### Get Delivery Status

**GET** `/status/{notification_id}`

```json
{
  "id": "uuid",
  "status": "delivered|failed|pending|bounced",
  "channel": "email",
  "delivered_at": "2026-06-22T10:31:10Z",
  "error": null
}
```

### List Configured Channels

**GET** `/channels`

```json
{
  "channels": [
    {
      "name": "primary-email",
      "type": "email",
      "active": true
    },
    {
      "name": "alerts-telegram",
      "type": "telegram",
      "active": true
    }
  ]
}
```

### Health Check

**GET** `/health`

```json
{
  "status": "ok",
  "database": "connected",
  "version": "1.1.3",
  "uptime_seconds": 3600
}
```

---

## Database Schema

All tables are prefixed with `np_notify_` and automatically include multi-tenant support:

| Table | Rows | Purpose |
|-------|------|---------|
| `np_notify_notifications` | ~1M | Individual notifications sent/pending |
| `np_notify_log` | ~10M | Detailed delivery log (kept for 90 days) |
| `np_notify_channels` | ~100 | Named channels (email, Telegram, SMS, etc.) |
| `np_notify_templates` | ~50 | Reusable message templates |
| `np_notify_inbox` | ~100K | Inbound webhook events from delivery services |
| `np_notify_retry_queue` | ~1K | Notifications pending retry |
| `np_notify_unsubscribes` | ~100K | User unsubscribe records per channel |
| `np_notify_webpush_subscriptions` | ~50K | Web push device subscriptions (browsers) |
| `np_notify_webhooks` | ~50 | Legacy: registered webhook endpoints |

**Multi-Tenant Columns (All Tables)**
- `tenant_id UUID` — Cloud tenant identifier (SaaS isolation)
- `source_account_id TEXT` — Multi-app account (default: `'primary'`)

Both enable safe data isolation in multi-tenant and multi-app deployments.

---

## Multi-Tenant Security

### Row-Level Security (RLS)

All queries are automatically scoped by the requesting tenant via Hasura RLS:

```sql
-- Hasura filter (active for cloud_user role)
WHERE tenant_id = X-Hasura-Tenant-Id
```

**This means:**
- User A (tenant-uuid-1) cannot see User B's (tenant-uuid-2) notifications
- All API responses are filtered by `X-Hasura-Tenant-Id` HTTP header
- Database constraints prevent cross-tenant reads/writes

### Verification

```bash
# As tenant 'abc-123':
curl -H "X-Hasura-Tenant-Id: abc-123" \
  http://localhost:3712/api/notifications

# Sees only abc-123's notifications. Other tenants' data is invisible.
```

---

## Troubleshooting

### Plugin won't install

```bash
# Verify license is active
nself license check

# Check plugin registry
nself plugin list notify

# Manual health check
curl http://localhost:3712/health
```

### Emails not sending

```bash
# Verify SMTP config
nself plugin config notify

# Check logs
nself plugin logs notify | tail -50

# Manually test SMTP
docker run --rm -e SMTP_HOST=mail.example.com \
  nself/plugin-notify:latest --test-smtp
```

### Missing notifications in database

```bash
# Check database connection
psql $DATABASE_URL -c "SELECT COUNT(*) FROM np_notify_notifications;"

# Check app can reach notify plugin
curl http://localhost:3712/health

# Review logs for permission errors
nself plugin logs notify | grep -i "error\|permission"
```

### Telegram not delivering

```bash
# Verify bot token is valid
curl -s https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getMe

# Check channel is active
curl http://localhost:3712/channels | jq '.channels[] | select(.type == "telegram")'
```

---

## Performance & Scaling

### Throughput
- Designed for ~100 notifications/sec per instance
- Retries follow exponential backoff (1s → 32s max)
- Batch endpoint scales to 1000 recipients/request

### Database Sizing
- `np_notify_log` grows ~50MB/day at 100 req/sec
- Automatic cleanup: entries older than 90 days are deleted
- Run `VACUUM ANALYZE np_notify_*` weekly on production

### Rate Limiting
- Per-channel limit: 10 msg/min by default (configurable)
- Per-recipient: 100 msg/day (prevents spam)
- Burst capacity: 50 queued, older requests dropped

---

## Integration Examples

### From ɳClaw

```go
// Send urgent notification to owner
resp, err := http.Post("http://notify:3712/send", "application/json",
  bytes.NewBufferString(`{
    "channel": "telegram",
    "recipient": "123456789",
    "title": "Claw Alert",
    "body": "Your deployment failed",
    "priority": "high"
  }`))
```

### From ɳChat

```typescript
// Alert team when message requires moderation
fetch('http://notify:3712/send', {
  method: 'POST',
  body: JSON.stringify({
    channel: 'slack',
    recipients: '#moderation',
    title: 'Flagged Message',
    body: `User ${userId} posted potentially harmful content.`,
    priority: 'normal'
  })
})
```

### From ClawDE

```python
# Notify user of analysis completion
import requests
requests.post('http://notify:3712/send', json={
    'channel': 'email',
    'recipient': user_email,
    'template_name': 'analysis-complete',
    'template_vars': {
        'analysis_id': analysis_id,
        'insights_count': 42
    },
    'priority': 'normal'
})
```

---

## License & Pricing

- **License:** Source-Available (ɳSelf proprietary)
- **Tier:** Pro
- **Pricing:** Included in ɳSelf+ ($3.99/mo or $39.99/yr)
- **Standalone:** Not sold separately; part of notify bundle

---

## Support & Updates

- **Changelog:** [CHANGELOG.md](../CHANGELOG.md)
- **Repository:** https://github.com/nself-org/plugins-pro
- **Issues:** Reported via `nself plugin report-issue notify`
- **Updates:** Auto-installed via `nself plugin update`

---

## References

- **Multi-Tenant Architecture:** `.claude/docs/architecture/multi-tenant-conventions.md`
- **Hasura RLS Setup:** `.github/docs/hasura-rls-configuration.md` (in plugin repo)
- **SSRF Guard:** All external URLs validated; configured in `internal/ssrf/guard.go`
- **Health Checks:** Weekly automated health checks via `nself plugin doctor`
