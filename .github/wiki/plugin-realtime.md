# Realtime Plugin

> WebSocket server with presence, typing indicators, rooms, and channels. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install realtime
```

## What It Does

Adds a production-grade WebSocket server to your ɳSelf stack. Clients connect and subscribe to named channels or rooms. The server broadcasts presence events (who's online), typing indicators, and custom events. Redis pub/sub handles fan-out across multiple server instances. Use it to add real-time features to any application without building WebSocket infrastructure.

## Dependencies

Requires Redis (`REDIS_ENABLED=true`).

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `REALTIME_PORT` | `3109` | Realtime service port |
| `REALTIME_MAX_CONNECTIONS` | `10000` | Max concurrent WebSocket connections |
| `REALTIME_PING_INTERVAL` | `30s` | WebSocket ping interval |
| `REALTIME_PRESENCE_TTL` | `60` | Presence expiry in seconds |

## Ports

| Port | Purpose |
|------|---------|
| 3109 | Realtime WebSocket server |

## Database Tables

6 tables added to your Postgres database:
- `np_realtime_rooms`, room definitions
- `np_realtime_channels`, channel definitions
- `np_realtime_presence`, current presence state
- `np_realtime_subscriptions`, active subscriptions
- `np_realtime_events`, event log
- `np_realtime_connections`, connection audit log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/realtime/` | Realtime REST API |
| `/realtime/ws` | WebSocket endpoint |
