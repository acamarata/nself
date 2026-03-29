# Content Progress Plugin

> Playback progress tracking — continue watching, watchlists, and favorites. **Free — MIT licensed.**

## Install

```bash
nself plugin install content-progress
```

## What It Does

Tracks where users left off in video content, maintains watchlists and favorites, and provides a "continue watching" feed. Progress is stored per-user per-content item with position and completion status. Works alongside the content-acquisition and torrent-manager plugins.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CONTENT_PROGRESS_PORT` | `3022` | Service port |
| `CONTENT_PROGRESS_RESUME_THRESHOLD` | `0.95` | Completion threshold (0-1) |

## Ports

| Port | Purpose |
|------|---------|
| 3022 | Content progress REST API |

## Database Tables

5 tables added to your Postgres database:
- `np_content_progress_progress` — per-user playback position
- `np_content_progress_watchlist` — user watchlists
- `np_content_progress_favorites` — user favorites
- `np_content_progress_history` — view history
- `np_content_progress_ratings` — user ratings

## Nginx Routes

| Route | Target |
|-------|--------|
| `/content-progress/` | Content progress API |
