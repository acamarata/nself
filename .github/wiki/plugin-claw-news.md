# Claw News Plugin

> News aggregation, AI summarization, and alerting for ɳClaw. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install claw-news
```

## What It Does

Polls RSS/Atom feeds, classifies articles by topic, runs AI summarization and sentiment analysis, and fires breaking-news alerts through the `notify` plugin. Generates scheduled digests on demand or via cron. Exposes a Claw tool descriptor so the agent can query articles, manage sources, and trigger digests conversationally.

## Dependencies

Requires the `ai` plugin. Optional: `notify` for breaking-news delivery.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CLAW_NEWS_PORT` | `3718` | Claw News service port |
| `PLUGIN_AI_INTERNAL_URL` | `http://plugin-ai:3709` | AI plugin URL for summarization |
| `PLUGIN_NOTIFY_URL` | `http://plugin-notify:3715` | Notify plugin URL for alerts |

## Ports

| Port | Purpose |
|------|---------|
| 3718 | REST API + Claw tool endpoint |

## Database Tables

4 tables added to your Postgres database:
- `np_news_sources` — registered RSS/Atom feeds
- `np_news_articles` — ingested articles with summaries
- `np_news_topics` — topic keywords and classifiers
- `np_news_alerts` — breaking-news alert rules

## Webhooks

| Event | Description |
|-------|-------------|
| `article.created` | New article ingested from a feed |
| `article.summarized` | AI summary generated for an article |
| `alert.fired` | Breaking-news alert triggered |
| `digest.generated` | Scheduled digest produced |

## Actions

| Action | Description |
|--------|-------------|
| `articles` | List and search ingested articles |
| `sources` | Add, update, or remove feed sources |
| `topics` | Manage topic keywords and classifiers |
| `alerts` | Configure breaking-news alerts |
| `digest` | Generate an AI news digest on demand |

## Capabilities

- RSS/Atom polling with per-source intervals
- AI summarization and sentiment scoring
- Topic classification using configurable keyword sets
- Breaking-news alerting via the `notify` plugin
- Digest generation (daily/weekly or ad-hoc)
- Claw tool descriptor — the ɳClaw agent can invoke news actions directly

## Multi-Tenant

Supported via the `source_account_id` isolation column. Each app in a shared backend sees only its own sources, articles, topics, and alerts.

## Health Check

`GET /health` — returns 200 when the ingestion loop and database connection are healthy.

← [[Plugin-Overview]] | [[Home]] →
