> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# SSO Plugin

> Single sign-on via SAML and OIDC for enterprise identity provider integration. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install sso
```

## What It Does

Adds enterprise SSO capabilities to your nSelf Auth configuration. Integrates with identity providers via SAML 2.0 and OpenID Connect (OIDC). Users from your organization's IdP (Okta, Azure AD, Google Workspace, etc.) can sign in to your application without a separate nSelf account. Maps IdP groups to nSelf roles automatically.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SSO_PORT` | `3054` | SSO service port |
| `SSO_DEFAULT_PROTOCOL` | `oidc` | Default protocol: `oidc` or `saml` |
| `SSO_SESSION_DURATION` | `8h` | SSO session duration |
| `SSO_ROLE_MAPPING_ENABLED` | `true` | Map IdP groups to nSelf roles |

## Ports

| Port | Purpose |
|------|---------|
| 3054 | SSO management REST API |

## Database Tables

5 tables added to your Postgres database:
- `np_sso_providers` — IdP provider configurations
- `np_sso_sessions` — active SSO sessions
- `np_sso_mappings` — group-to-role mappings
- `np_sso_assertions` — SAML assertion log
- `np_sso_tokens` — OIDC token cache

## Nginx Routes

| Route | Target |
|-------|--------|
| `/sso/` | SSO callback and management API |
| `/sso/callback` | IdP callback endpoint |
