# Post Plugin

> Multi-platform content publishing with optional scheduling. Credentials encrypted with AES-256-GCM at rest. **Pro plugin — requires ɳClaw bundle or ɳSelf+ license.**

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install post
nself build
```

## What It Does

The post plugin publishes content to multiple platforms from a single queue. Posts can be scheduled, retried on failure, or published immediately. Platform credentials are encrypted with AES-256-GCM before storage in `np_post_accounts` — do not set them as plain environment variables.

ɳClaw uses this plugin automatically when asked to publish content. Say "post this to my blog" and ɳClaw calls the plugin directly.

## Supported Platforms

| Platform | Status | Auth method |
|----------|--------|-------------|
| WordPress | Working | Application Password (XML-RPC) |
| Ghost | Working | Admin API key (JWT HS256) |
| Twitter/X | Working | OAuth 1.0a user-context or Bearer Token |
| LinkedIn | Working | OAuth 2.0 access token (UGC Posts API) |
| Telegram | Working | Bot API token + chat ID |
| Dev.to | Working | API key (Forem Articles API) |
| Hashnode | Working | Personal Access Token (GraphQL) |
| Mastodon | Planned | — |
| Bluesky | Planned | — |
| Facebook | Planned | — |

## Per-Platform Setup

### WordPress

Generate an Application Password under **Users → Profile → Application Passwords** in wp-admin. Use that password (not your login password) as `app_password`.

```bash
curl -X POST http://localhost:3129/post/accounts \
  -H "X-Internal-Secret: $POST_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "wordpress",
    "name": "My Blog",
    "credentials": {
      "endpoint": "https://blog.example.com",
      "username": "editor",
      "app_password": "xxxx xxxx xxxx xxxx xxxx xxxx"
    }
  }'
```

### Ghost

Generate an Admin API key under **Settings → Integrations** in Ghost Admin. The key format is `<keyId>:<hexSecret>`.

```bash
curl -X POST http://localhost:3129/post/accounts \
  -H "X-Internal-Secret: $POST_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "ghost",
    "name": "My Ghost Blog",
    "credentials": {
      "endpoint": "https://blog.example.com",
      "admin_api_key": "6423e2f2e7a5bbde0e0e6cd8:1f33a7..."
    }
  }'
```

### Twitter/X

Create a Twitter Developer App with read and write permissions. Generate OAuth 1.0a user tokens under **Keys and Tokens** in the developer portal.

```bash
curl -X POST http://localhost:3129/post/accounts \
  -H "X-Internal-Secret: $POST_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "twitter",
    "name": "My Twitter",
    "credentials": {
      "oauth_consumer_key": "...",
      "oauth_consumer_secret": "...",
      "oauth_token": "...",
      "oauth_token_secret": "..."
    }
  }'
```

### LinkedIn

Obtain an OAuth 2.0 access token with `w_member_social` scope via the `linkedin` plugin's OAuth flow. Find your member URN in the LinkedIn API using your profile ID.

```bash
curl -X POST http://localhost:3129/post/accounts \
  -H "X-Internal-Secret: $POST_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "linkedin",
    "name": "My LinkedIn",
    "credentials": {
      "access_token": "AQV...",
      "author_urn": "urn:li:person:ABC123"
    }
  }'
```

### Telegram

Create a bot via @BotFather and add it to your target channel or group with posting permissions. The `chat_id` is the channel or group ID (use `@channelusername` for public channels).

```bash
curl -X POST http://localhost:3129/post/accounts \
  -H "X-Internal-Secret: $POST_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "telegram",
    "name": "My Channel",
    "credentials": {
      "bot_token": "123456789:AAF...",
      "chat_id": "@mychannel",
      "parse_mode": "HTML"
    }
  }'
```

### Dev.to

Generate an API key at https://dev.to/settings/extensions.

```bash
curl -X POST http://localhost:3129/post/accounts \
  -H "X-Internal-Secret: $POST_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "devto",
    "name": "My Dev.to",
    "credentials": {
      "api_key": "a1b2c3d4..."
    }
  }'
```

### Hashnode

Generate a Personal Access Token at https://hashnode.com/settings/developer. Find your publication ID under General Settings in your publication dashboard.

```bash
curl -X POST http://localhost:3129/post/accounts \
  -H "X-Internal-Secret: $POST_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "hashnode",
    "name": "My Hashnode Blog",
    "credentials": {
      "token": "hn_PAT...",
      "publication_id": "abc123def456"
    }
  }'
```

## Publishing

Publish immediately to a connected account:

```bash
curl -X POST http://localhost:3129/post/publish \
  -H "X-Internal-Secret: $POST_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "acc_abc123",
    "title": "My post title",
    "content": "Post body here. Markdown or HTML depending on platform.",
    "tags": ["tag1", "tag2"]
  }'
```

Schedule a post for later:

```bash
curl -X POST http://localhost:3129/post/publish \
  -H "X-Internal-Secret: $POST_INTERNAL_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "acc_abc123",
    "title": "Scheduled post",
    "content": "Coming soon...",
    "schedule_at": "2026-07-01T09:00:00Z"
  }'
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `POST_PORT` | `3129` | Port the service listens on |
| `POST_INTERNAL_SECRET` | — | Shared secret for all API calls |
| `POST_ENCRYPTION_KEY` | — | 32-byte AES-256 key for encrypting credentials |
| `DATABASE_URL` | — | PostgreSQL connection string |

## Ports

| Port | Purpose |
|------|---------|
| `3129` | Post plugin REST API |

## Database Tables

Two tables are added to your Postgres database:

- `np_post_accounts`: connected platform accounts with encrypted credentials
- `np_post_queue`: post queue with status, schedule time, and dispatch result

Both tables use `source_account_id` for multi-app isolation.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/post/` | `localhost:3129` |

---

See also: [[Plugin-Overview]] | [[plugin-linkedin]] | [[Home]]
