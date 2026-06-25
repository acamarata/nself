# Admin API Plugin

> Aggregated metrics, system health, session counts, and storage breakdown for your ɳSelf instance. **Pro plugin.**

> **Requires:** Pro license tier. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install admin-api
```

## What It Does

Provides admin-only REST endpoints for monitoring a running ɳSelf instance. It collects point-in-time metric snapshots (CPU, memory, active connections, request counts), reports per-service health, stores dashboard configuration, and exposes aggregated dashboard statistics over a 24-hour window. The Admin GUI at `localhost:3021` consumes these endpoints, but any admin client can call them directly.

## Authentication

Authentication is optional and activates only when `ADMIN_API_KEY` is set. When set, every request to the `/api/v1/*` routes must carry the key in either:

- `X-Api-Key: <key>`, or
- `Authorization: Bearer <key>`

The `/health` and `/ready` probes are always unauthenticated. Hasura is not used for this surface, so no Hasura admin role is required, the plugin talks to Postgres directly with `DATABASE_URL`.

## Endpoints

| Method | Route | Purpose |
|--------|-------|---------|
| GET | `/health` | Liveness probe (unauthenticated) |
| GET | `/api/v1/metrics` | Latest metric values |
| GET | `/api/v1/metrics/history` | Historical metric snapshots |
| POST | `/api/v1/metrics/snapshot` | Record a new metric snapshot |
| GET | `/api/v1/health/services` | Per-service health overview |
| GET | `/api/v1/stats` | Aggregated dashboard statistics (24h) |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ADMIN_API_PLUGIN_PORT` | `3212` | Admin API service port |
| `DATABASE_URL` | (none) | PostgreSQL connection URL (required) |
| `ADMIN_API_KEY` | (none) | When set, enables `X-Api-Key` / `Bearer` auth |
| `PROMETHEUS_URL` | (none) | Optional Prometheus endpoint for metric scraping |
| `ADMIN_API_CACHE_TTL` | `30` | Cache TTL in seconds |
| `ADMIN_API_WS_ENABLED` | `false` | Enable WebSocket support |
| `CORS_ALLOWED_ORIGIN` | `http://localhost:3021` | Allowed CORS origin |

## Ports

| Port | Purpose |
|------|---------|
| 3212 | Admin API REST endpoints |

**Port note:** Port 3212 was historically shared with the `hipaa` plugin. That conflict was resolved during the P4-E7 port truth-up: `hipaa` moved to **3214**, and `admin-api` keeps **3212** (per SPORT `F10-PORT-REGISTRY.md`). If you run an older `hipaa` build still pinned to 3212, set `ADMIN_API_PLUGIN_PORT` or upgrade `hipaa` to clear the collision.

## Database Tables

Three tables are created in your Postgres database, each carrying `source_account_id` for multi-app isolation:

- `np_admin_metrics_snapshots`, point-in-time metric records
- `np_admin_dashboard_config`, dashboard configuration key/value store
- `np_admin_audit_log`, admin action audit trail

## Nginx Routes

| Route | Target | Notes |
|-------|--------|-------|
| `/admin-api/` | Admin API | Restricted, authentication required when `ADMIN_API_KEY` is set |

---

[[Plugin-Overview]] · [[Home]]
