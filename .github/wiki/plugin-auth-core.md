# plugin-auth — Core Reference

**Port:** 9006 | **Bundle:** ClawDE + ɳChat | **License:** required | **Language:** Go

The auth plugin provides an 18-flow authentication suite as a standalone HTTP service. It runs on port 9006 alongside your nSelf stack and handles all authentication concerns server-side: JWT signing, bcrypt hashing, MFA secrets, OAuth state, and session management never touch your application code.

## Installation

```bash
nself license activate <your-license-key>
nself plugin install auth
```

Start:

```bash
nself plugin start auth
```

Verify:

```bash
curl http://localhost:9006/health
# {"status":"ok"}
```

## Auth Flows (18 total)

| # | Flow | Description |
|---|------|-------------|
| 1 | Sign in | Bcrypt password verify → JWT access + refresh token |
| 2 | Sign up | Create user + bcrypt hash + email verification |
| 3 | Sign out | Revoke current session + bump token_version |
| 4 | Refresh token | Rotate JWT pair using valid refresh token |
| 5 | Verify password | Check current password before sensitive changes |
| 6 | Change password | Re-hash + invalidate all other sessions |
| 7 | Password reset | Request + confirm flow (SHA-256 token, single-use) |
| 8 | Email verification | Verify + resend (enum-safe: always 200) |
| 9 | Magic links | Send + verify (email-based passwordless) |
| 10 | TOTP 2FA | Enroll → QR → verify + backup codes |
| 11 | WebAuthn/Passkeys | Register + authenticate (discoverable + non-discoverable) |
| 12 | OAuth (10 providers) | Authorize + callback; PKCE S256 for public clients |
| 13 | Device code | RFC 8628 device flow for TV/headless clients |
| 14 | ID.me | Identity verification (veteran, student, healthcare) |
| 15 | SAML/SSO | Enterprise SSO with SAML 2.0 (IdP-initiated + SP-initiated) |
| 16 | Session management | List, create, rotate, revoke, revoke-all, activity |
| 17 | Trusted devices | Skip 2FA on known devices |
| 18 | Registration lock | Admin-controlled user registration gate |

## Required Configuration

```bash
# AES-256 key for MFA secret encryption — REQUIRED
MFA_ENCRYPTION_KEY=<base64-encoded 32-byte key>

# Generate:
openssl rand -base64 32

# Database
DATABASE_URL=postgresql://user:pass@host:5432/db

# JWT signing
JWT_SECRET=<secret>
```

The plugin refuses to start if `MFA_ENCRYPTION_KEY` is missing or decodes to fewer than 32 bytes.

## OAuth Providers

Ten providers supported. Set the relevant env vars; providers without credentials are silently skipped.

| Provider | Env vars |
|----------|----------|
| Google | `AUTH_GOOGLE_CLIENT_ID`, `AUTH_GOOGLE_CLIENT_SECRET` |
| Apple | `AUTH_APPLE_CLIENT_ID`, `AUTH_APPLE_TEAM_ID`, `AUTH_APPLE_KEY_ID`, `AUTH_APPLE_PRIVATE_KEY` |
| Facebook | `AUTH_FACEBOOK_CLIENT_ID`, `AUTH_FACEBOOK_CLIENT_SECRET` |
| GitHub | `AUTH_GITHUB_CLIENT_ID`, `AUTH_GITHUB_CLIENT_SECRET` |
| Microsoft | `AUTH_MICROSOFT_CLIENT_ID`, `AUTH_MICROSOFT_CLIENT_SECRET`, `AUTH_MICROSOFT_TENANT` |
| Discord | `AUTH_DISCORD_CLIENT_ID`, `AUTH_DISCORD_CLIENT_SECRET` |
| Twitter/X | `AUTH_TWITTER_CLIENT_ID`, `AUTH_TWITTER_CLIENT_SECRET` |
| LinkedIn | `AUTH_LINKEDIN_CLIENT_ID`, `AUTH_LINKEDIN_CLIENT_SECRET` |
| GitLab | `AUTH_GITLAB_CLIENT_ID`, `AUTH_GITLAB_CLIENT_SECRET` |
| Slack | `AUTH_SLACK_CLIENT_ID`, `AUTH_SLACK_CLIENT_SECRET` |

OAuth initiate flow: `GET /api/v1/oauth/{provider}/authorize` — redirects to provider.  
OAuth callback: `POST /api/v1/oauth/{provider}/callback` — exchanges code, JIT-links provider to user.

## WebAuthn / Passkeys

```bash
WEBAUTHN_ENABLED=true
WEBAUTHN_RP_ID=yourdomain.com     # bare hostname, no scheme/port
WEBAUTHN_RP_NAME=YourApp
```

Production deployments must set `WEBAUTHN_RP_ID` to the public domain. `localhost` is valid for local dev only.

## TOTP 2FA

```bash
AUTH_TOTP_ISSUER=YourApp          # shown in authenticator app
AUTH_TOTP_DIGITS=6
AUTH_TOTP_PERIOD=30
AUTH_TOTP_BACKUP_CODE_COUNT=10    # single-use codes per enrollment
```

## Session Policy

```bash
AUTH_SESSION_MAX_PER_USER=10
AUTH_SESSION_IDLE_TIMEOUT_HOURS=720       # 30 days
AUTH_SESSION_ABSOLUTE_TIMEOUT_HOURS=8760  # 365 days
AUTH_LOGIN_MAX_ATTEMPTS=5
AUTH_LOGIN_LOCKOUT_MINUTES=15
```

## API Quick Reference

### Core

```
POST   /api/v1/auth/signin
POST   /api/v1/auth/signup
POST   /api/v1/auth/signout
POST   /api/v1/auth/refresh
POST   /api/v1/auth/verify-password
POST   /api/v1/auth/change-password
POST   /api/v1/auth/password-reset       (request)
PUT    /api/v1/auth/password-reset       (confirm)
POST   /api/v1/auth/verify-email
POST   /api/v1/auth/resend-verification
```

### Sessions

```
GET    /api/v1/sessions
POST   /api/v1/sessions
PUT    /api/v1/sessions                  (rotate)
DELETE /api/v1/sessions                  (revoke all)
GET    /api/v1/sessions/activity
DELETE /api/v1/sessions/{id}
```

### MFA

```
GET    /api/v1/mfa/enrollments
POST   /api/v1/mfa/enrollments
DELETE /api/v1/mfa/enrollments/{id}
POST   /api/v1/mfa/verify
GET    /api/v1/mfa/backup-codes
POST   /api/v1/mfa/backup-codes/redeem
```

### Passkeys

```
GET    /api/v1/passkeys
POST   /api/v1/passkeys
DELETE /api/v1/passkeys/{id}
POST   /api/v1/passkeys/verify
```

### OAuth

```
GET    /api/v1/oauth/providers
POST   /api/v1/oauth/providers
PUT    /api/v1/oauth/providers/{id}
DELETE /api/v1/oauth/providers/{id}
GET    /api/v1/oauth/{provider}/authorize
POST   /api/v1/oauth/{provider}/callback
```

### Magic Links

```
POST   /api/v1/magic-links/send
POST   /api/v1/magic-links/verify
```

### SSO / SAML

```
GET    /api/v1/sso
POST   /api/v1/sso
GET    /api/v1/sso/{id}
PUT    /api/v1/sso/{id}
DELETE /api/v1/sso/{id}
GET    /api/v1/sso/{id}/metadata
POST   /api/v1/sso/{id}/test
GET    /api/v1/sso/login
POST   /api/v1/sso/callback
POST   /api/v1/sso/slo
```

### ID.me

```
GET    /api/v1/idme/authorize
POST   /api/v1/idme/callback
POST   /api/v1/idme/verify
GET    /api/v1/idme/status
POST   /api/v1/idme/revoke
```

### Trusted Devices & Registration Lock

```
GET    /api/v1/trusted-devices
POST   /api/v1/trusted-devices
DELETE /api/v1/trusted-devices

GET    /api/v1/registration-lock
POST   /api/v1/registration-lock
POST   /api/v1/registration-lock/verify
DELETE /api/v1/registration-lock
```

### Observability

```
GET    /api/v1/login-attempts
GET    /api/v1/stats
GET    /health
GET    /ready
```

## Database Tables

All tables use `np_auth_` prefix and include `tenant_id UUID` for Hasura row-level security.

| Table | Purpose |
|-------|---------|
| `np_auth_users` | User records with bcrypt password hash |
| `np_auth_sessions` | Active session tokens with device info |
| `np_auth_login_attempts` | Audit log for rate limiting + lockout |
| `np_auth_oauth_providers` | Configured OAuth provider credentials |
| `np_auth_oauth_states` | PKCE + CSRF state nonces (single-use) |
| `np_auth_passkeys` | WebAuthn credential descriptors |
| `np_auth_mfa_enrollments` | TOTP secrets (AES-256-GCM encrypted) |
| `np_auth_device_codes` | RFC 8628 device authorization codes |
| `np_auth_magic_links` | SHA-256-hashed passwordless tokens |
| `np_auth_trusted_devices` | Trusted device fingerprints |
| `np_auth_registration_locks` | Per-tenant registration gate config |
| `np_auth_password_resets` | SHA-256-hashed reset tokens (single-use) |
| `np_auth_email_verifications` | Email confirmation tokens |
| `np_auth_idme_verifications` | ID.me identity verification records |
| `np_auth_sso_connections` | SAML IdP connection metadata |
| `np_auth_sso_states` | SAML AuthnRequest nonces |

## Hasura RLS

Every `np_auth_*` table has a Hasura row filter enforcing `tenant_id = X-Hasura-Tenant-Id`. This is applied automatically by `nself plugin install auth` via the `hasura_metadata` section in `plugin.json`.

Example filter:

```json
{
  "np_auth_sessions": {
    "filter": { "tenant_id": { "_eq": "X-Hasura-Tenant-Id" } }
  }
}
```

## Security Notes

- **MFA secrets** are AES-256-GCM encrypted in Postgres. The `MFA_ENCRYPTION_KEY` must be backed up separately.
- **Passwords** use bcrypt cost 12. Constant-time comparison prevents timing attacks.
- **Tokens** (reset, magic-link, email-verify) are stored as SHA-256 hashes only. The raw token is sent once and never stored.
- **Sign out everywhere** bumps `token_version` in `np_auth_users`. Any JWT with an older `tv` claim is rejected on next verification.
- **Rate limiting** is per-identifier (email/username) on sign-in and verify-password. After `AUTH_LOGIN_MAX_ATTEMPTS` failures within the window, the account is locked for `AUTH_LOGIN_LOCKOUT_MINUTES`.
- **Email enumeration protection**: password-reset and resend-verification always return HTTP 200 regardless of whether the address exists.

## Testing

```bash
# Unit tests (no external deps)
cd plugins-pro/paid/auth
go test ./... -v

# Integration tests (requires Docker stack with Mailhog + OAuth mock)
go test ./go/... -v -tags integration -timeout 120s \
  -run TestAuthSuite \
  TEST_AUTH_URL=http://localhost:4000 \
  TEST_MAIL_CATCHER_URL=http://localhost:8025
```

## Webhooks

17 webhook events. Configure the webhook endpoint via `plugin.json` → `config.webhookPath` (default: `/webhooks/auth`).

Key events: `auth.login.success`, `auth.login.failure`, `auth.login.blocked`, `auth.session.created`, `auth.session.revoked`, `auth.mfa.enrolled`, `auth.passkey.registered`, `auth.oauth.linked`, `auth.magic_link.used`, `auth.device_code.authorized`.

## Related

- `OAUTH-SETUP.md` — per-provider OAuth credential setup guide
- `plugins-pro/paid/auth/migrations/` — database schema SQL
- `cli/.github/wiki/plugin-auth.md` — CLI command reference for auth plugin
- `cli/.github/wiki/auth-flow.md` — auth flow diagrams
