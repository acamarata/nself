# Auth Core Plugin

> 18-flow authentication suite: sessions, OAuth (10 providers), WebAuthn/passkeys, TOTP 2FA, magic links, device-code flow, SSO/SAML, ID.me, registration locks, MFA backup codes. **Pro plugin.**

> **Requires:** Pro license (ClawDE or ɳChat bundle). `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install auth
```

## What It Does

Provides a complete authentication backend for ɳSelf apps. Handles all 18 auth flows:

| Flow | Description |
|------|-------------|
| Sign up / Sign in | Email+password with bcrypt hashing |
| Sign out | Session revocation |
| Refresh token | JWT refresh with rotation |
| Verify password | Credential re-confirmation |
| Change password | Authenticated password update |
| Password reset | Request + confirm via email token |
| Email verification | Email confirm + resend |
| Sessions | Create / rotate / list / activity / revoke-all |
| Trusted devices | Register, list, and remove trusted devices |
| Registration lock | Enable, verify, and disable sign-up pin lock |
| MFA backup codes | List and redeem single-use backup codes |
| OAuth providers | CRUD for linked social accounts |
| OAuth authorize/callback | Generic + per-provider initiate+callback for 10 social providers |
| Passkeys (WebAuthn) | Register, list, verify, and delete FIDO2 passkeys |
| MFA enrollments | TOTP enroll / verify / delete |
| Magic links | Send and verify one-time login links |
| Device code flow | Initiate, poll, and authorize headless device logins |
| SSO/SAML | Connection CRUD + login, callback, and SLO |
| ID.me | Government identity verification (authorize, callback, verify, status, revoke) |

## Configuration

### Required

| Env Var | Description |
|---------|-------------|
| `MFA_ENCRYPTION_KEY` | AES-256 key for TOTP secrets, base64-encoded 32 bytes minimum. Generate: `openssl rand -base64 32` |
| `DATABASE_URL` | Postgres connection URL |

### Optional — Core

| Env Var | Default | Description |
|---------|---------|-------------|
| `AUTH_PLUGIN_PORT` | `9006` | HTTP listening port |
| `AUTH_PLUGIN_HOST` | `` | Bind host (default: all interfaces) |
| `AUTH_LOG_LEVEL` | `info` | Log verbosity |
| `JWT_SECRET` | | JWT signing secret |

### Optional — OAuth Providers

| Env Var | Provider |
|---------|----------|
| `AUTH_GOOGLE_CLIENT_ID` + `AUTH_GOOGLE_CLIENT_SECRET` | Google |
| `AUTH_APPLE_CLIENT_ID` + `AUTH_APPLE_TEAM_ID` + `AUTH_APPLE_KEY_ID` + `AUTH_APPLE_PRIVATE_KEY` | Apple |
| `AUTH_FACEBOOK_APP_ID` + `AUTH_FACEBOOK_APP_SECRET` | Facebook |
| `AUTH_GITHUB_CLIENT_ID` + `AUTH_GITHUB_CLIENT_SECRET` | GitHub |
| `AUTH_MICROSOFT_CLIENT_ID` + `AUTH_MICROSOFT_CLIENT_SECRET` + `AUTH_MICROSOFT_TENANT` | Microsoft |
| `AUTH_DISCORD_CLIENT_ID` + `AUTH_DISCORD_CLIENT_SECRET` | Discord |
| `AUTH_TWITTER_CLIENT_ID` + `AUTH_TWITTER_CLIENT_SECRET` | Twitter/X |

### Optional — WebAuthn / Passkeys

| Env Var | Default | Description |
|---------|---------|-------------|
| `WEBAUTHN_ENABLED` | `false` | Enable WebAuthn / passkey flows |
| `WEBAUTHN_RP_ID` | `localhost` | Relying Party ID — bare hostname, no scheme |
| `WEBAUTHN_RP_NAME` | `nSelf` | Relying Party display name shown in browser dialog |
| `WEBAUTHN_RP_ORIGINS` | `http://localhost:9006` | Comma-separated allowed origins (canonical) |
| `AUTH_WEBAUTHN_RP_ORIGINS` | | Legacy alias for `WEBAUTHN_RP_ORIGINS` |

### Optional — TOTP / MFA

| Env Var | Default | Description |
|---------|---------|-------------|
| `AUTH_TOTP_ISSUER` | `nself` | TOTP issuer name in authenticator app |
| `AUTH_TOTP_DIGITS` | `6` | TOTP code length (6 or 8) |
| `AUTH_TOTP_PERIOD` | `30` | TOTP rotation period in seconds |
| `AUTH_TOTP_BACKUP_CODE_COUNT` | `10` | Number of one-time backup codes generated on TOTP enroll |

### Optional — Session / Login Policy

| Env Var | Default | Description |
|---------|---------|-------------|
| `AUTH_SESSION_MAX_PER_USER` | `10` | Max concurrent sessions per user |
| `AUTH_SESSION_IDLE_TIMEOUT_HOURS` | | Session idle expiry in hours |
| `AUTH_SESSION_ABSOLUTE_TIMEOUT_HOURS` | | Session hard expiry in hours |
| `AUTH_LOGIN_MAX_ATTEMPTS` | `5` | Failed logins before lockout |
| `AUTH_LOGIN_LOCKOUT_MINUTES` | `15` | Lockout duration in minutes |

## Ports

| Port | Purpose |
|------|---------|
| `9006` | Auth HTTP API (all 18 flows) |

## Database Tables

16 tables added to your Postgres database, all scoped by `source_account_id` (Multi-App Isolation):

| Table | Purpose |
|-------|---------|
| `auth_users` | User accounts (email, hashed password, verification state) |
| `auth_sessions` | JWT session records with device metadata |
| `auth_oauth_providers` | Linked OAuth accounts per user |
| `auth_oauth_states` | CSRF nonces for OAuth state param |
| `auth_passkeys` | WebAuthn credential records |
| `auth_mfa_enrollments` | TOTP enrollment secrets (AES-256-GCM encrypted) |
| `auth_device_codes` | Device-code flow state |
| `auth_magic_links` | One-time magic link tokens |
| `auth_login_attempts` | Failed login audit log |
| `auth_password_resets` | Password reset tokens |
| `auth_email_verifications` | Email verification tokens |
| `auth_trusted_devices` | Remembered trusted device fingerprints |
| `auth_registration_locks` | Per-user sign-up lock pins |
| `auth_idme_verifications` | ID.me government identity results |
| `auth_sso_connections` | SAML/OIDC SSO provider configurations |
| `auth_sso_states` | SSO CSRF nonces |

All tables use `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation (Convention A). Hasura row-filters use `X-Hasura-Source-Account-Id` on all roles.

## Hasura Integration

The plugin registers 16 tables in Hasura with row-level security enforced via `source_account_id`. All select and insert permissions are filtered through `{"source_account_id": {"_eq": "X-Hasura-Source-Account-Id"}}`.

## Key API Routes

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/v1/auth/signup` | public |
| POST | `/api/v1/auth/signin` | public |
| POST | `/api/v1/auth/signout` | bearer |
| POST | `/api/v1/auth/refresh` | public |
| POST | `/api/v1/auth/password-reset` | public |
| PUT | `/api/v1/auth/password-reset` | public |
| POST | `/api/v1/auth/verify-email` | public |
| GET | `/api/v1/sessions` | bearer |
| DELETE | `/api/v1/sessions/{id}` | bearer |
| POST | `/api/v1/passkeys` | bearer |
| POST | `/api/v1/passkeys/verify` | bearer |
| POST | `/api/v1/mfa/enrollments` | bearer |
| POST | `/api/v1/mfa/verify` | bearer |
| POST | `/api/v1/magic-links/send` | bearer |
| POST | `/api/v1/magic-links/verify` | bearer |
| GET | `/api/v1/oauth/{provider}/authorize` | public |
| POST | `/api/v1/oauth/{provider}/callback` | public |
| GET | `/api/v1/idme/authorize` | public |
| GET | `/api/v1/sso` | bearer |
| POST | `/api/v1/sso/{id}/test` | bearer |
| GET | `/health` | public |

Full route list: 48 routes across all 18 flows. See [API Reference](API-Reference.md).

## Bundles

Included in:
- **ClawDE** bundle (`$0.99/mo`)
- **ɳChat** bundle (`$0.99/mo`)
- **ɳSelf+** (`$3.99/mo` — all bundles)

## Security Notes

- All TOTP secrets are stored AES-256-GCM encrypted in the database. `MFA_ENCRYPTION_KEY` must be a 32-byte base64-encoded secret and is validated at startup — the service refuses to start without it.
- WebAuthn flows require `WEBAUTHN_ENABLED=true` and an HTTPS origin in production. `WEBAUTHN_RP_ID` must be a bare hostname (no scheme, port, or path).
- Login lockout is applied per `(source_account_id, email)` pair.
- Backup codes are single-use and hashed at rest; they cannot be retrieved after generation.
- RLS policy prevents cross-app data access at the database row level via `source_account_id`.

## CLI Actions

```bash
nself plugin auth init        # Initialize auth database schema
nself plugin auth server      # Start auth HTTP server on port 9006
nself plugin auth sessions    # List active sessions for a user
nself plugin auth revoke-session  # Revoke a specific session
nself plugin auth revoke-all  # Revoke all sessions for a user
nself plugin auth mfa-status  # Check MFA enrollment status
nself plugin auth login-attempts  # View recent login attempts
nself plugin auth oauth-connections  # List OAuth connections
nself plugin auth cleanup-expired  # Clean up expired tokens and codes
nself plugin auth stats       # Show auth plugin statistics
```
