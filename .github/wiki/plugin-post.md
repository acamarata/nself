# Post Plugin

> Social media posting automation for multiple platforms. Credentials encrypted with AES-256-GCM at rest. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install post
```

## What It Does

The post plugin automates social media publishing across Twitter/X, LinkedIn, Mastodon, and Bluesky from a single queue. Platform credentials are encrypted with AES-256-GCM before being stored, and posts can be scheduled, retried on failure, or fanned out to multiple accounts simultaneously.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `POST_PORT` | `3129` | Port the service listens on |

Supported platforms: Twitter/X, LinkedIn, Mastodon, Bluesky. Per-account credentials are stored encrypted in `np_post_accounts` and managed via the API — do not set them as plain env vars.

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
