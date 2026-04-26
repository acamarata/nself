# Podcast Plugin

> RSS feed management, episode hosting, and multi-device playback sync. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install podcast
```

## What It Does

A complete podcast backend for hosting and distributing podcast episodes. Manages episode uploads to MinIO, generates standards-compliant RSS feeds for Apple Podcasts and Spotify, tracks per-device playback progress for cross-device sync, and provides chapter marker support.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PODCAST_PORT` | `3210` | Podcast service port |
| `PODCAST_STORAGE_BUCKET` | — | MinIO bucket for audio files |
| `PODCAST_BASE_URL` | — | Public base URL for episode URLs in RSS |
| `PODCAST_DEFAULT_LANGUAGE` | `en` | Default podcast language |

## Ports

| Port | Purpose |
|------|---------|
| 3210 | Podcast REST API |

## Database Tables

5 tables added to your Postgres database:
- `np_podcast_shows`, podcast show definitions
- `np_podcast_episodes`, episode records
- `np_podcast_chapters`, episode chapter markers
- `np_podcast_progress`, per-device playback progress
- `np_podcast_subscriptions`, subscriber records

## Nginx Routes

| Route | Target |
|-------|--------|
| `/podcast/` | Podcast management API |
| `/podcast/{show_id}/feed.xml` | RSS feed for distribution |
