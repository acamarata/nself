# Feature Flags Plugin

> Feature flag service with targeting rules, user segments, and evaluation engine. **Free, MIT licensed.**

## Install

```bash
nself plugin install feature-flags
```

## What It Does

Enables toggling features per user, tenant, or custom segment without redeploying. Supports percentage rollouts, user targeting rules, and environment-specific flags. Evaluation happens server-side; the REST API serves flag decisions to your application.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `FEATURE_FLAGS_PORT` | `3305` | Feature flags service port |
| `FEATURE_FLAGS_CACHE_TTL` | `60` | Flag evaluation cache TTL (seconds) |

## Ports

| Port | Purpose |
|------|---------|
| 3305 | Feature flags REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_feature_flags_flags`, flag definitions and targeting rules
- `np_feature_flags_segments`, user segment definitions

## Nginx Routes

| Route | Target |
|-------|--------|
| `/feature-flags/` | Feature flags API |

## API

```
GET  /health               — Health check
GET  /flags                — List all flags
POST /flags                — Create a flag
PUT  /flags/{key}          — Update flag rules
GET  /evaluate/{key}       — Evaluate flag for a user context
POST /evaluate/batch       — Evaluate multiple flags at once
```
