# Admin API Plugin

> System metrics, service health, session management, and audit log API for your nSelf instance. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install admin-api
```

## What It Does

Provides admin-only REST endpoints for monitoring and managing a running nSelf instance. Exposes service health checks, real-time system metrics, active session enumeration and revocation, and a queryable audit log. All endpoints require the `ADMIN_API_SECRET` header and are restricted at the Nginx layer.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ADMIN_API_PORT` | `3212` | Admin API service port |
| `ADMIN_API_SECRET` | — | Required bearer secret for all admin endpoints |
| `ADMIN_SESSION_RETENTION` | `90` | Number of days to retain session records |

## Ports

| Port | Purpose |
|------|---------|
| 3212 | Admin API REST endpoints |

## Database Tables

2 tables added to your Postgres database:
- `np_admin_api_sessions` — tracked admin session records
- `np_admin_api_audit_log` — admin action audit trail

## Nginx Routes

| Route | Target | Notes |
|-------|--------|-------|
| `/admin-api/` | Admin API | Restricted — authentication required |
