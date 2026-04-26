# Post Plugin

> Multi-platform content publishing with optional scheduling. Credentials encrypted with AES-256-GCM at rest. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install post
```

## What It Does

The post plugin publishes content to multiple platforms from a single queue. Posts can be scheduled, retried on failure, or published simultaneously to several destinations. Platform credentials are encrypted with AES-256-GCM before being stored in `np_post_accounts`, do not set them as plain env vars.

## Supported Platforms

| Platform | Status |
|----------|--------|
| WordPress | Working |
| Ghost | Working |
| Telegram | Working |
| Dev.to | Working |
| Hashnode | Working |
| Twitter/X | Planned (v1.1.x) |
| LinkedIn | Planned (v1.1.x) |
| Mastodon | Planned (v1.1.x) |
| Bluesky | Planned (v1.1.x) |
| Facebook | Planned (v1.1.x) |
| Instagram | Planned (v1.1.x) |
| Threads | Planned (v1.1.x) |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `POST_PORT` | `3129` | Port the service listens on |

## Ports

| Port | Purpose |
|------|---------|
| `3129` | Post automation REST API |

## Database Tables

2 tables added to your Postgres database:

- `np_post_accounts`
- `np_post_queue`

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/post/` | `localhost:3129` |
