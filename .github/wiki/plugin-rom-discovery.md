# ROM Discovery Plugin

> ROM database search, automated discovery, scraper job queue, hash-based deduplication. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install rom-discovery
```

## What It Does

The rom-discovery plugin scans configured sources for ROMs, computes hashes for deduplication, and enqueues scraper jobs to match files against known ROM databases. Duplicate detection prevents the same game from appearing multiple times across different dump sources.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ROM_DISCOVERY_PORT` | `3125` | Port the service listens on |
| `ROM_STORAGE_PATH` | — | Filesystem path scanned for ROMs |

## Ports

| Port | Purpose |
|------|---------|
| `3125` | ROM discovery REST API |

## Database Tables

4 tables added to your Postgres database:

- `np_rom_discovery_roms`
- `np_rom_discovery_sources`
- `np_rom_discovery_jobs`
- `np_rom_discovery_hashes`

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/rom-discovery/` | `localhost:3125` |
