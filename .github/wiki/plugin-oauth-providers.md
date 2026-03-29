> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# OAuth Providers Plugin

> Additional OAuth providers beyond nSelf defaults — enterprise IdP support. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install oauth-providers
```

## What It Does

Extends nSelf Auth with additional OAuth 2.0 provider integrations beyond the built-in Google, GitHub, and Apple. Adds enterprise providers including Microsoft Azure AD, Okta, Auth0, Slack, LinkedIn, Twitter/X, and Discord. Each provider is configured via environment variables and integrates with the nSelf Auth session system.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `OAUTH_PROVIDERS_PORT` | `3057` | OAuth providers service port |
| `AZURE_CLIENT_ID` | — | Microsoft Azure AD client ID |
| `AZURE_CLIENT_SECRET` | — | Microsoft Azure AD client secret |
| `AZURE_TENANT_ID` | — | Azure AD tenant ID |
| `OKTA_CLIENT_ID` | — | Okta client ID |
| `OKTA_CLIENT_SECRET` | — | Okta client secret |
| `OKTA_DOMAIN` | — | Your Okta domain |
| `SLACK_CLIENT_ID` | — | Slack app client ID |
| `SLACK_CLIENT_SECRET` | — | Slack app client secret |
| `DISCORD_CLIENT_ID` | — | Discord app client ID |
| `DISCORD_CLIENT_SECRET` | — | Discord app client secret |

## Ports

| Port | Purpose |
|------|---------|
| 3057 | OAuth provider management API |

## Database Tables

3 tables added to your Postgres database:
- `np_oauth_providers_configs` — provider configurations
- `np_oauth_providers_tokens` — user OAuth token cache
- `np_oauth_providers_links` — user-to-provider account links

## Nginx Routes

| Route | Target |
|-------|--------|
| `/oauth/{provider}/callback` | OAuth callback endpoints |
