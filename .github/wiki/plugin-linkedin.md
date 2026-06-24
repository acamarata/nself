# LinkedIn Plugin

> LinkedIn publishing integration with OAuth 2.0 and post history. **Pro plugin. Requires license.**

## Bundle membership

This plugin is part of the **ɳClaw bundle** ($0.99/mo or $9.99/yr). It is also included in **ɳSelf+** ($3.99/mo or $39.99/yr), which covers all 6 bundles and all apps.

| Option | Price | Includes |
|--------|-------|---------|
| ɳClaw bundle | $0.99/mo / $9.99/yr | All ɳClaw plugins including linkedin |
| ɳSelf+ | $3.99/mo / $39.99/yr | All bundles + all apps |
| Free | $0 | CLI + 29 free plugins (linkedin not included) |

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install linkedin
nself build
```

The license is validated against `ping.nself.org/license/validate`. Tier is checked server-side; insufficient tier returns an error.

## Description

The LinkedIn plugin connects a LinkedIn account via OAuth 2.0, posts to a member's LinkedIn feed with optional image attachments, and tracks post history for later reference. It also exposes a Claw tool descriptor so ɳClaw can publish on the user's behalf.

OAuth tokens are stored per `source_account_id`, so multi-tenant ɳSelf installs keep each user's LinkedIn credentials isolated. The plugin is currently in `beta` status.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | — | Postgres connection string (required) |
| `LINKEDIN_CLIENT_ID` | — | LinkedIn OAuth app client ID (required) |
| `LINKEDIN_CLIENT_SECRET` | — | LinkedIn OAuth app client secret (required) |
| `LINKEDIN_REDIRECT_URI` | — | OAuth callback URL (required) |
| `LINKEDIN_INTERNAL_SECRET` | — | Internal request signing secret (required) |
| `PORT` | `3722` | LinkedIn service port |
| `BIND_ADDRESS` | `127.0.0.1` | Bind address for the service |
| `NSELF_PLUGIN_LICENSE_KEY` | — | License key (read from CLI by default) |

## Ports

| Port | Purpose |
|------|---------|
| 3722 | LinkedIn service REST API |

## Database Schema

2 tables added to your Postgres database (prefix: `np_linkedin_`):

- `np_linkedin_tokens`, OAuth access and refresh tokens per account
- `np_linkedin_posts`, published post history with status

## REST API

```
GET  /health                       — Health check
GET  /oauth/start                  — Begin OAuth connection
GET  /oauth/callback               — Complete OAuth handshake
POST /posts                        — Publish a post to the connected feed
GET  /posts                        — List published posts
```

## Examples

### Publish a text post

```bash
curl -X POST http://localhost:3722/posts \
  -H "Content-Type: application/json" \
  -d '{"text":"New ɳSelf release shipping today.","visibility":"PUBLIC"}'
```

### List recent posts

```bash
curl http://localhost:3722/posts
```

## Source

Source-available (license required to run): [`plugins-pro/paid/linkedin/`](https://github.com/nself-org/plugins-pro/tree/main/paid/linkedin)

Note: `plugins-pro` is a private repository. Source access is granted to ɳSelf+ subscribers and Enterprise customers.

## See Also

- [[plugin-post]], multi-platform publisher (includes LinkedIn as a target)
- [[plugin-google]], similar OAuth-driven integration pattern
- [[Plugin-Licensing]], tier comparison
- [[Plugin-Overview]], full plugin index

← [[Plugin-Overview]] | [[Home]] →
