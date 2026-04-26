# Link Preview Plugin

> URL metadata extraction, Open Graph, Twitter Card, and oEmbed with caching. **Free, MIT licensed.**

## Install

```bash
nself plugin install link-preview
```

## What It Does

Extracts rich metadata from URLs for link preview cards. Fetches Open Graph tags, Twitter Card metadata, and oEmbed data. Results are cached in Postgres to avoid repeated fetches. Use it to power link preview features in chat, CMS, or social feed applications.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `LINK_PREVIEW_PORT` | `3718` | Link preview service port |
| `LINK_PREVIEW_CACHE_TTL` | `3600` | Cache TTL in seconds (1 hour) |
| `LINK_PREVIEW_TIMEOUT` | `10s` | HTTP fetch timeout |
| `LINK_PREVIEW_MAX_SIZE` | `5MB` | Max page size to fetch |

## Ports

| Port | Purpose |
|------|---------|
| 3718 | Link preview REST API |

## Database Tables

1 table added to your Postgres database:
- `np_link_preview_cache`, cached URL metadata (title, description, image, favicon)

## Nginx Routes

| Route | Target |
|-------|--------|
| `/link-preview/` | Link preview API |

## API

```
GET  /health                    — Health check
GET  /preview?url={url}         — Fetch or return cached preview
DELETE /preview?url={url}       — Invalidate cache for URL
GET  /preview/batch             — Batch preview multiple URLs
```
