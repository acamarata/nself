# Feature: Auth

nSelf includes nHost Auth as a built-in service, providing a full authentication backend out of the box.

## What's Included

| Capability | Description |
|-----------|-------------|
| JWT tokens | Short-lived access tokens + refresh tokens |
| OAuth 2.0 | GitHub, Google, Facebook, Apple, and more |
| Magic links | Passwordless email login |
| Email/password | Standard username/password auth |
| MFA | Time-based one-time passwords (TOTP) |
| User management | Roles, metadata, session management |

## How to Enable

Auth is a core service — it is always enabled. No plugin required.

## Key Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `AUTH_JWT_SECRET` | JWT signing secret | Auto-generated |
| `AUTH_CLIENT_URL` | Allowed redirect URL after login | Required |
| `AUTH_SERVER_URL` | Auth service public URL | Derived from BASE_DOMAIN |
| `AUTH_RATE_LIMIT` | Auth endpoint rate limit | `30r/m` |
| `AUTH_SMTP_HOST` | SMTP host for magic link emails | Optional |

## OAuth Provider Setup

To enable OAuth providers, set the relevant env vars:

```env
AUTH_PROVIDER_GITHUB_ENABLED=true
AUTH_PROVIDER_GITHUB_CLIENT_ID=your-client-id
AUTH_PROVIDER_GITHUB_CLIENT_SECRET=your-secret
```

## See Also

- [[Config-Auth]] — full auth configuration reference
- [[cmd-service]] — enable/disable auth
- [[Feature-Plugins]] — extend auth with the pro auth plugin

---
← [[Home]] | [[_Sidebar]]
