# Geolocation Plugin

> Real-time location sharing, history tracking, and haversine geofencing with entry/exit events. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install geolocation
```

## What It Does

The geolocation plugin enables real-time location sharing between users and records a configurable-retention history of position updates. Geofences trigger entry and exit events that downstream services can subscribe to, with haversine calculations used for accurate distance-based boundary checks.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `GEOLOCATION_PORT` | `3026` | Port the service listens on |
| `GEOLOCATION_RETENTION_DAYS` | `30` | Number of days to retain location history |

## Ports

| Port | Purpose |
|------|---------|
| `3026` | Geolocation REST API |

## Database Tables

5 tables added to your Postgres database:

- `np_geolocation_users`
- `np_geolocation_locations`
- `np_geolocation_history`
- `np_geolocation_geofences`
- `np_geolocation_events`

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/geolocation/` | `localhost:3026` |
