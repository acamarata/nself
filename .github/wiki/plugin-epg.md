# EPG Plugin

> Electronic Program Guide — XMLTV import, channels, schedules, and recording rules. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install epg
```

## What It Does

Manages an Electronic Program Guide for IPTV and broadcast TV applications. Imports channel and schedule data from XMLTV sources, maintains a current/upcoming program schedule, and manages recording rules that trigger the `recording` plugin. Provides a GraphQL API via Hasura for guide display in client apps.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `EPG_PORT` | `3090` | EPG service port |
| `EPG_XMLTV_URL` | — | XMLTV source URL |
| `EPG_REFRESH_INTERVAL` | `86400` | Schedule refresh interval in seconds |
| `EPG_SCHEDULE_DAYS` | `7` | Days of schedule to maintain |

## Ports

| Port | Purpose |
|------|---------|
| 3090 | EPG REST API |

## Database Tables

6 tables added to your Postgres database:
- `np_epg_sources` — XMLTV source configurations
- `np_epg_channels` — channel definitions
- `np_epg_programs` — program schedule
- `np_epg_categories` — program categories
- `np_epg_recording_rules` — scheduled recording rules
- `np_epg_sync_log` — XMLTV sync history

## Nginx Routes

| Route | Target |
|-------|--------|
| `/epg/` | EPG query API |
