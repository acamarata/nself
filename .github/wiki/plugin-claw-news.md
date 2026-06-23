# ɳClaw News Plugin

**Pro plugin for ɳClaw bundle** — RSS/Atom feed aggregation, AI summarization, and breaking-news alerting.

## Overview

The claw-news plugin turns ɳClaw into a personal news aggregator. It polls RSS and Atom feeds on a schedule, ingests articles into PostgreSQL, runs each through the `ai` plugin for automated summarization and topic classification, and fires breaking-news alerts via the optional `notify` plugin.

Users interact with claw-news as a tool within ɳClaw: "What's new in AI today?" queries the article store directly, combining keyword search with sentiment and importance scoring. Daily digest jobs roll up the most relevant articles per topic into AI-generated summaries.

## Tier and Pricing

| Tier | Monthly | Annual | Includes claw-news? |
|------|---------|--------|-----|
| Free | $0 | $0 | No |
| Any bundle | $0.99/mo | $9.99/yr | ✓ (ɳClaw Bundle) |
| ɳSelf+ | $3.99/mo | $39.99/yr | ✓ |

**Minimum tier:** claw-news requires any paid bundle (tier `pro`). Install requires a valid `nself_pro_*` license key.

## Install

### Prerequisites
- nSelf CLI v1.0.0+
- Valid `nself_pro_*` license key (purchase at `nself.org/pricing`)
- Paid tier subscription (ɳClaw Bundle or ɳSelf+)
- `ai` plugin (required for summarization)
- `notify` plugin (optional, for breaking-news alerts)

### Steps

```bash
# 1. Set your license (if not already set)
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 2. Install claw-news
nself plugin install claw-news

# 3. Build the local stack
nself build

# 4. Start services
nself start
```

The license is validated against `ping.nself.org/license/validate` during build. Insufficient tier returns an error; valid license proceeds.

## Configuration

Configure via environment variables (set in `.env` or pass to `nself build`):

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string (e.g. `postgres://user:pass@localhost:5432/nself`) |
| `PORT` | No | `3718` | Listen port for the plugin service |
| `PLUGIN_AI_INTERNAL_URL` | No | `http://plugin-ai:3401` | Internal URL of the `ai` plugin (required for summarization) |
| `PLUGIN_NOTIFY_URL` | No | — | Internal URL of the `notify` plugin (required for breaking-news alerts) |

Ports are registered in the nSelf port registry. Port 3718 is fixed for claw-news; override only for testing.

## Architecture

### Data Model

Tables (all prefixed `np_news_`, tenant-isolated via `tenant_id`):

| Table | Purpose |
|-------|---------|
| `np_news_sources` | RSS/Atom feed definitions (name, URL, poll interval, last fetch timestamp) |
| `np_news_articles` | Ingested articles (title, URL, raw content, AI summary, topics, sentiment, importance score) |
| `np_news_topics` | User-defined alert topics (keywords, source filters, importance thresholds) |
| `np_news_alerts` | Fired alert records (topic, channel, delivery status) |
| `np_news_content_hashes` | Content deduplication table (prevent duplicate articles from different sources) |

All tables include `tenant_id` for multi-tenancy and `source_account_id` for multi-app isolation within a tenant.

### Polling Loop

- Initial poll: 5 seconds after startup
- Subsequent polls: every 30 seconds, checking each source against its `poll_interval_secs` (default 900s = 15 minutes)
- Feed fetch timeout: 30 seconds per feed
- Payload limit: 10 MB per feed

### Feed Security (SSRF Guard)

All feed URLs are validated before fetch via `ssrf.ValidateURL()`:
- **Blocked:** RFC 1918 private ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`)
- **Blocked:** loopback (`127.0.0.0/8`, `::1`)
- **Blocked:** link-local and metadata endpoints (`169.254.0.0/16`, AWS/GCP/Azure/Alibaba metadata IPs)
- **Blocked:** non-http(s) schemes
- **Blocked:** hostnames containing "localhost", "metadata", or "internal"

This ensures feeds registered by users cannot access internal services or metadata endpoints.

## REST API

All endpoints require authentication (JWT bearer token). Base path: `http://localhost:3718/`

### Articles

- **GET `/articles`** — List all articles (filter by topic, date, source)
  - Query params: `?topic=<id>&date_after=<ISO8601>&source=<id>&limit=50&offset=0`
  - Response: `{articles: [{id, title, url, summary, sentiment, importance, published_at}, ...], total: N}`

- **GET `/articles/:id`** — Get one article with full content
  - Response: `{id, title, url, content, ai_summary, topics, sentiment, importance, published_at, created_at}`

### Sources

- **GET `/sources`** — List all configured RSS sources
  - Response: `{sources: [{id, name, url, type, poll_interval_secs, last_polled_at, enabled}, ...]}`

- **POST `/sources`** — Register a new RSS/Atom feed
  - Body: `{name: "Hacker News", url: "https://hnrss.org/frontpage", type: "rss", poll_interval_secs: 600, enabled: true}`
  - Response: `{id: "...", name, url, ...}`

- **DELETE `/sources/:id`** — Remove a source
  - Response: `{success: true}`

### Topics & Alerts

- **GET `/topics`** — List user-defined alert topics
  - Response: `{topics: [{id, name, keywords: [...], min_importance, enabled, created_at}, ...]}`

- **POST `/topics`** — Create an alert topic
  - Body: `{name: "AI Safety", keywords: ["AI regulation", "AI safety"], min_importance: 0.5, enabled: true}`
  - Response: `{id: "...", name, keywords, ...}`

- **POST `/digest`** — Generate an AI digest
  - Body: `{period: "24h", max_articles: 20, topic_id: "..."}`
  - Response: `{content: "# News Digest\n\n## AI Safety\n...", generated_at: "ISO8601"}`

- **GET `/alerts`** — List recent alerts
  - Query params: `?limit=50&offset=0`
  - Response: `{alerts: [{id, topic_id, topic_name, article_id, channel, fired_at}, ...]}`

### Health & Status

- **GET `/health`** — Liveness probe
  - Response: `{status: "ok"}`

## Webhooks

When events occur, claw-news emits webhook events (if subscribed via Hasura event triggers):

| Event | When | Payload |
|-------|------|---------|
| `article.created` | New article ingested | `{article_id, source_id, title, url, published_at}` |
| `article.summarized` | AI summary generated | `{article_id, summary, topics, sentiment, importance}` |
| `alert.fired` | Breaking alert triggered | `{alert_id, topic_id, article_id, channel, fired_at}` |
| `digest.generated` | Daily digest completed | `{digest_id, topic_id, content, generated_at}` |

## Examples

### Add a feed

```bash
curl -X POST http://localhost:3718/sources \
  -H "Authorization: Bearer $BEARER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Hacker News",
    "url": "https://hnrss.org/frontpage",
    "type": "rss",
    "poll_interval_secs": 600,
    "enabled": true
  }'
```

### Create a breaking-alert topic

```bash
curl -X POST http://localhost:3718/topics \
  -H "Authorization: Bearer $BEARER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "AI regulation",
    "keywords": ["AI regulation", "AI safety act", "EU AI"],
    "min_importance": 0.6,
    "enabled": true
  }'
```

### Query articles for a topic

```bash
curl -X GET 'http://localhost:3718/articles?topic=<topic_id>&limit=10' \
  -H "Authorization: Bearer $BEARER_TOKEN"
```

### Generate today's digest

```bash
curl -X POST http://localhost:3718/digest \
  -H "Authorization: Bearer $BEARER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "period": "24h",
    "max_articles": 20
  }'
```

## Multi-Tenancy

claw-news enforces tenant isolation via Hasura row-level security (RLS). All operations are automatically scoped to the tenant of the authenticated user (determined by the `X-Hasura-Tenant-Id` JWT claim).

- User cannot see articles, sources, or alerts from other tenants
- Digest and alert jobs run per-tenant
- No manual tenant ID in requests needed — Hasura enforces automatically

## Dependencies

- **Requires:** `ai` plugin (for summarization, topic classification, importance scoring)
- **Optional:** `notify` plugin (for breaking-news alerts via push, SMS, Slack, Telegram)
- **Optional:** `cron` plugin (for scheduling custom digest jobs)

If `ai` plugin is unavailable, articles are ingested but summaries and topics are not generated until the plugin is available.

If `notify` plugin is unavailable, alerts are recorded but not delivered; re-attempt delivery on next plugin availability.

## Security

- **SSRF Guard:** All feed URLs validated against RFC 1918 and metadata endpoints before fetch
- **RLS:** All data scoped to tenant via Hasura row filters
- **License Gate:** Invalid or expired license blocks install and operation
- **Content Integrity:** Deduplication via content hash prevents re-ingesting identical articles

## Performance

- Feed fetch timeout: 30 seconds per source
- Poll interval: user-configurable (default 15 minutes)
- Daily digest: async, background job
- Full-text search: indexed on article title and content for sub-second lookup
- Importance scoring: cached and recomputed on article update

## Troubleshooting

### Plugin fails to start

**Symptom:** `nself start` fails, claw-news service doesn't boot.

**Check:**
1. `DATABASE_URL` is set and PostgreSQL is reachable: `psql $DATABASE_URL -c "SELECT 1"`
2. Migrations ran successfully: check `np_news_sources` table exists
3. `ai` plugin is running: `curl http://plugin-ai:3401/health`

**Fix:**
```bash
nself plugin uninstall claw-news
nself plugin install claw-news
nself build
nself start
```

### Articles not being fetched

**Symptom:** Feed sources are configured but no articles appear.

**Check:**
1. Source is enabled: `SELECT enabled FROM np_news_sources WHERE id = '...'`
2. Feed URL is valid: `curl -u '' <feed_url>` returns 200 and valid RSS/Atom
3. No SSRF block: check logs for "SSRF check failed"
4. Poller is running: check service logs `nself logs claw-news` | grep "poller"

**Fix:**
- If SSRF block: feed URL may be private/internal. Allowlist via `NSELF_SSRF_ALLOWLIST` env var (CIDR notation)
- If DNS fails: verify feed hostname resolves: `nslookup <hostname>`

### Summaries not being generated

**Symptom:** Articles ingested but `ai_summary` is NULL.

**Check:**
1. `ai` plugin is running: `curl http://plugin-ai:3401/health`
2. Articles have raw content: `SELECT content FROM np_news_articles LIMIT 1`
3. `PLUGIN_AI_INTERNAL_URL` is correct

**Fix:**
```bash
nself plugin install ai
nself build
nself start
# Summaries auto-generate on next poll
```

### License invalid

**Symptom:** `nself build` fails with "license validation failed" or "tier does not include claw-news".

**Check:**
1. License key is set: `nself license get`
2. License is valid for this machine: visit `nself.org/account` and check license status
3. Subscription includes ɳClaw Bundle or ɳSelf+

**Fix:**
```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself build
```

## See Also

- **ɳClaw assistant** (`nclaw` repo) — the AI assistant that queries claw-news
- **AI plugin** — required for summarization and topic classification
- **Notify plugin** — optional, for breaking-news delivery
- **Cron plugin** — useful for scheduling custom digest jobs
- **Bundle membership** — see `.github/docs/licensing/bundles.md`
- **License validation** — see `.github/docs/licensing/ping-api.md`
- **Architecture** — see `Architecture.md` in this wiki

---

## Changelog

### v1.0.0 (2026-06-22)
- Initial release: RSS/Atom polling, AI summarization, topic classification, breaking alerts
- Multi-tenant support via Hasura RLS
- SSRF guard blocks RFC 1918 and metadata endpoints
- Integration with `ai` and `notify` plugins
