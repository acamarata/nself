# LinkedIn Plugin

> LinkedIn publishing integration. OAuth 2.0, post to feed with optional images, post history, Claw tool descriptor. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`
> **Status:** beta

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install linkedin
```

## What It Does

Authenticates users via LinkedIn OAuth 2.0 and publishes posts to their feed with optional image attachments. Stores tokens per account and keeps a local post-history log. Exposes a Claw tool descriptor so the ɳClaw agent can compose and publish posts conversationally.

## Configuration

| Env Var | Required | Description |
|---------|----------|-------------|
| `DATABASE_URL` | Yes | Postgres connection string |
| `LINKEDIN_CLIENT_ID` | Yes | LinkedIn OAuth app client ID |
| `LINKEDIN_CLIENT_SECRET` | Yes | LinkedIn OAuth app client secret |
| `LINKEDIN_REDIRECT_URI` | Yes | OAuth callback URL registered with LinkedIn |
| `LINKEDIN_INTERNAL_SECRET` | Yes | Shared secret for internal API calls |
| `PORT` | No | Override default port (3722) |
| `BIND_ADDRESS` | No | Override bind address (default `127.0.0.1`) |
| `NSELF_PLUGIN_LICENSE_KEY` | No | License key (usually inherited from global) |

## Ports

| Port | Purpose |
|------|---------|
| 3722 | REST API + OAuth callback handler |

## Database Tables

2 tables added to your Postgres database:
- `np_linkedin_tokens` — per-account OAuth tokens (encrypted at rest)
- `np_linkedin_posts` — published post history

## Capabilities

- LinkedIn OAuth 2.0 (authorization code flow)
- Publish text posts with optional image attachment
- Per-account token storage with refresh
- Post-history log for audit and analytics
- Claw tool descriptor — ɳClaw can compose and publish posts

## Multi-Tenant

Supported via the `source_account_id` isolation column. Each app in a shared backend sees only its own tokens and post history.

## Setup: LinkedIn OAuth App

1. Create an OAuth app at <https://www.linkedin.com/developers/apps>
2. Configure the redirect URI to match `LINKEDIN_REDIRECT_URI`
3. Request the `w_member_social` scope for publishing
4. Copy the Client ID and Client Secret into your `.env.secrets`

## Health Check

`GET /health` — returns 200 when the database and LinkedIn API reachability are healthy.

← [[Plugin-Overview]] | [[Home]] →
