# Content Acquisition Plugin

> RSS monitoring, release calendar, and automated download pipeline. **Free — MIT licensed.**

## Install

```bash
nself plugin install content-acquisition
```

## What It Does

Monitors RSS feeds and release calendars for new content, then automatically triggers download pipelines. Stores feed state, discovered items, and download jobs in Postgres. Integrates with the torrent-manager and subtitle-manager plugins to form a complete content pipeline.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CONTENT_ACQUISITION_PORT` | `3202` | Service port |
| `CONTENT_ACQUISITION_POLL_INTERVAL` | `300` | Feed poll interval in seconds |
| `CONTENT_ACQUISITION_DOWNLOAD_DIR` | `/downloads` | Base download directory |

## Ports

| Port | Purpose |
|------|---------|
| 3202 | Content acquisition REST API |

## Database Tables

8 tables added to your Postgres database:
- `np_content_acquisition_feeds` — RSS/Atom feed subscriptions
- `np_content_acquisition_feed_items` — discovered feed items
- `np_content_acquisition_releases` — release calendar entries
- `np_content_acquisition_download_jobs` — queued downloads
- `np_content_acquisition_download_history` — completed downloads
- `np_content_acquisition_filters` — content filtering rules
- `np_content_acquisition_categories` — content categories
- `np_content_acquisition_indexers` — indexer configurations

## Nginx Routes

| Route | Target |
|-------|--------|
| `/content-acquisition/` | Content acquisition API |
