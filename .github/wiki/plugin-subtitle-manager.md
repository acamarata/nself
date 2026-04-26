# Subtitle Manager Plugin

> Subtitle search and download via OpenSubtitles with transcoder integration. **Free, MIT licensed.**

## Install

```bash
nself plugin install subtitle-manager
```

## What It Does

Searches for and downloads subtitles for your media library via the OpenSubtitles API. Matches content by filename hash or IMDB ID. Stores downloaded subtitle files and tracks which languages are available per content item. Integrates with the content-acquisition plugin to automatically fetch subtitles for new downloads.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SUBTITLE_MANAGER_PORT` | `3204` | Service port |
| `SUBTITLE_MANAGER_API_KEY` | — | OpenSubtitles API key |
| `SUBTITLE_MANAGER_LANGUAGES` | `en` | Comma-separated preferred languages |
| `SUBTITLE_MANAGER_STORAGE_DIR` | `/subtitles` | Subtitle storage path |

## Ports

| Port | Purpose |
|------|---------|
| 3204 | Subtitle manager REST API |

## Database Tables

3 tables added to your Postgres database:
- `np_subtitle_manager_subtitles`, downloaded subtitle records
- `np_subtitle_manager_searches`, search history
- `np_subtitle_manager_language_prefs`, per-content language preferences

## Nginx Routes

None, subtitle service is internal only.
