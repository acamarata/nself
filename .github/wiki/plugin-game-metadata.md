# Game Metadata Plugin

> IGDB integration for game metadata. ROM hash matching, full-text search, artwork retrieval. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install game-metadata
```

## What It Does

The game-metadata plugin connects to the IGDB API to enrich your ROM library with titles, cover art, screenshots, and platform data. It matches ROMs by hash to known game entries and caches results locally for fast full-text search without repeated API calls.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `IGDB_CLIENT_ID` | — | Twitch/IGDB OAuth client ID |
| `IGDB_CLIENT_SECRET` | — | Twitch/IGDB OAuth client secret |
| `GAME_METADATA_PORT` | `3123` | Port the service listens on |

## Ports

| Port | Purpose |
|------|---------|
| `3123` | Game metadata REST API |

## Database Tables

5 tables added to your Postgres database:

- `np_game_metadata_games`
- `np_game_metadata_platforms`
- `np_game_metadata_artwork`
- `np_game_metadata_roms`
- `np_game_metadata_search_cache`

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/game-metadata/` | `localhost:3123` |
