# TMDB Plugin

> Movie/TV metadata from TMDB, IMDb, TVDB, and MusicBrainz with match queue and sync. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install tmdb
```

## What It Does

Enriches your media library with rich metadata from multiple sources: The Movie Database (TMDB), IMDb, TheTVDB, and MusicBrainz for music. Match media files by title or filename, bulk-import metadata for a library, keep metadata in sync with source updates, and search across all sources from a unified API. Used by IPTV and media server applications.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `TMDB_PORT` | `3122` | TMDB plugin port |
| `TMDB_API_KEY` | — | TMDB API v4 read access token |
| `TMDB_LANGUAGE` | `en-US` | Metadata language |
| `TMDB_SYNC_ENABLED` | `true` | Sync metadata updates periodically |
| `TMDB_SYNC_INTERVAL` | `86400` | Sync interval in seconds |

## Ports

| Port | Purpose |
|------|---------|
| 3122 | TMDB metadata REST API |

## Database Tables

7 tables added to your Postgres database:
- `np_tmdb_movies`, movie metadata
- `np_tmdb_tv_shows`, TV show metadata
- `np_tmdb_episodes`, TV episode metadata
- `np_tmdb_people`, cast and crew
- `np_tmdb_match_queue`, pending metadata match jobs
- `np_tmdb_artwork`, poster, backdrop, and logo references
- `np_tmdb_sync_log`, metadata sync history

## Nginx Routes

| Route | Target |
|-------|--------|
| `/tmdb/` | Metadata search and management API |
