# Retro Gaming Plugin

> ROM library management, emulator core integration, save state sync, controller configuration. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install retro-gaming
```

## What It Does

The retro-gaming plugin manages your self-hosted ROM library and emulator cores. It handles save state synchronisation across sessions, persists per-user controller configurations, and tracks achievements unlocked during play.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `RETRO_GAMING_PORT` | `3124` | Port the service listens on |
| `RETRO_STORAGE_PATH` | — | Filesystem path for ROM and save-state storage |

## Ports

| Port | Purpose |
|------|---------|
| `3124` | Retro gaming REST API |

## Database Tables

6 tables added to your Postgres database:

- `np_retro_gaming_roms`
- `np_retro_gaming_cores`
- `np_retro_gaming_saves`
- `np_retro_gaming_configs`
- `np_retro_gaming_sessions`
- `np_retro_gaming_achievements`

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/retro-gaming/` | `localhost:3124` |
