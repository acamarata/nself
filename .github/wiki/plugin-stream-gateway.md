# Stream Gateway Plugin

> Stream admission control — viewer authentication, analytics, and family controls. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install stream-gateway
```

## What It Does

Acts as a gatekeeper between viewers and your streaming server. Validates viewer entitlements before allowing stream access, tracks concurrent viewer counts, collects per-viewer quality analytics (buffering, bitrate switches), and enforces family member controls such as content ratings and concurrent stream limits per account.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `STREAM_GATEWAY_PORT` | `3093` | Stream gateway port |
| `STREAM_GATEWAY_MAX_STREAMS_PER_ACCOUNT` | `3` | Concurrent stream limit |
| `STREAM_GATEWAY_TOKEN_TTL` | `3600` | Viewer access token TTL |
| `STREAM_GATEWAY_GEO_ENABLED` | `false` | Enable geo-restriction |

## Ports

| Port | Purpose |
|------|---------|
| 3093 | Stream gateway REST API |

## Database Tables

4 tables added to your Postgres database:
- `np_stream_gateway_sessions` — active viewer sessions
- `np_stream_gateway_analytics` — per-viewer quality metrics
- `np_stream_gateway_profiles` — family member profiles
- `np_stream_gateway_restrictions` — content access restrictions

## Nginx Routes

| Route | Target |
|-------|--------|
| `/stream-gateway/` | Stream gateway API |
| `/stream-gateway/admit` | Viewer admission endpoint |
