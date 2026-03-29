# Geocoding Plugin

> Forward and reverse geocoding with haversine distance calculation and geofence storage and events. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install geocoding
```

## What It Does

The geocoding plugin provides forward geocoding (address to coordinates) and reverse geocoding (coordinates to address) through a pluggable provider backend. It stores geofences and emits entry and exit events when coordinates cross defined boundaries, with haversine distance calculations available for proximity queries.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `GEOCODING_PORT` | `3203` | Port the service listens on |
| `GEOCODING_PROVIDER` | `nominatim` | Geocoding provider: `nominatim`, `google`, or `mapbox` |
| `GEOCODING_API_KEY` | — | API key for Google or Mapbox providers (not required for Nominatim) |

## Ports

| Port | Purpose |
|------|---------|
| `3203` | Geocoding REST API |

## Database Tables

4 tables added to your Postgres database:

- `np_geocoding_addresses`
- `np_geocoding_geofences`
- `np_geocoding_cache`
- `np_geocoding_events`

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/geocoding/` | `localhost:3203` |
