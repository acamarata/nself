# Auth Flow — nSelf Unified Auth (D2)

This page documents the end-to-end authentication flows for `api.nself.org` powered by the auth plugin (D2-T01). Implementation is in `web/backend/plugins/auth/` and `web/backend/migrations/025_np_auth_plugin_schema.sql`.

For plugin install instructions see [plugin-auth.md](plugin-auth.md).

---

## Schema

Five Postgres tables back the auth system (migration `025_np_auth_plugin_schema.sql`):

| Table | Purpose |
|-------|---------|
| `np_users` | Primary identity records (email, hashed password, email_verified, source_account_id, tenant_id) |
| `np_sessions` | Server-side session store with revocation support |
| `np_mfa_devices` | TOTP enrollments and backup codes |
| `np_password_reset_tokens` | PKCE-style single-use reset tokens (15-min TTL) |
| `np_oauth_states` | CSRF state params for social login (10-min TTL) |

**Multi-Tenant Convention Wall (Decision #20):**
- `source_account_id TEXT NOT NULL DEFAULT 'primary'` — isolates self-hosted app instances
- `tenant_id UUID` nullable — Cloud customer isolation only
- Never swap these. Confusing them causes cross-customer data leaks.

---

## Signup Flow

> Status: **Implemented in D2-T02** (stub: email send stubbed until D4/Postmark is live)

```
POST /auth/signup  { email, password }
  → validate email format + password strength
  → check duplicate email (409 if exists)
  → rate-limit: 3 signups per email per hour
  → captcha verification (hCaptcha / Cloudflare Turnstile)
  → Argon2id hash password (time=3, mem=64MB, p=4)
  → INSERT np_users (email_verified = FALSE)
  → generate email verification JWT (15-min TTL)
  → send verification email via D4/Postmark (template: signup-verify)
  ← 201 { message: "Check your email to verify your account" }
```

**Security invariants:**
- Password never stored in plaintext or logged
- Duplicate email returns 409 with "email already registered" (no timing oracle)
- Unverified users cannot sign in (403 "check your email")

---

## Email Verification Flow

> Status: **Implemented in D2-T02**

```
GET /auth/verify-email?token=<signed JWT>
  → verify JWT signature + TTL (15 min)
  → verify token is single-use (used_at check)
  → UPDATE np_users SET email_verified = TRUE
  → mark token used
  ← 200 redirect to /account
```

**Re-send:**
```
POST /auth/resend-verify  { email }
  → rate-limit: 1 per 60s (dev: 1 per 5s)
  → generate new verification token
  → send email
  ← 200 { message: "Verification email sent" }
```

---

## Signin Flow

> Status: **Implemented in D2-T03** (stub: session cookie)

```
POST /auth/signin  { email, password }
  → check email_verified (403 if false)
  → check account lockout (403 if locked: 5 failed attempts → 15-min lockout)
  → Argon2id verify password
  → on success: INSERT np_sessions (expires_at = now + 7 days)
  → issue JWT (RS256, kid = AUTH_JWT_KEY_ID)
      payload: { sub: user_id, email, tenant_id?, source_account_id, exp, iat, session_id }
      Hasura claims namespace: https://hasura.io/jwt/claims
        X-Hasura-User-Id, X-Hasura-Default-Role, X-Hasura-Allowed-Roles
        X-Hasura-Source-Account-Id, X-Hasura-Tenant-Id (if Cloud)
  → set cookie: HttpOnly; SameSite=Lax; Secure; Path=/; Max-Age=604800; Domain=.nself.org
  ← 200 { user: { id, email, email_verified } }
```

---

## Password Reset Flow (PKCE — Amendment 9 SEC-1)

> Status: **Implemented in D2-T04**

```
POST /auth/forgot-password  { email }
  → always return 200 (no email enumeration)
  → generate code_verifier (32 random bytes, base64url)
  → code_challenge = BASE64URL(SHA256(code_verifier))
  → INSERT np_password_reset_tokens (code_challenge, expires_at = now + 15 min)
  → send reset email with link containing code_verifier
  ← 200 { message: "If that email is registered, a reset link was sent" }

POST /auth/reset-password  { code_verifier, new_password }
  → recompute code_challenge = BASE64URL(SHA256(code_verifier))
  → look up token WHERE code_challenge = $1 AND used_at IS NULL AND expires_at > now
  → validate (400 if not found, expired, or already used)
  → Argon2id hash new_password
  → UPDATE np_users SET hashed_password = new_hash
  → UPDATE np_password_reset_tokens SET used_at = NOW()
  → REVOKE ALL SESSIONS: UPDATE np_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL
  ← 200 { message: "Password reset. Please sign in with your new password." }
```

---

## MFA (TOTP) Flow

> Status: **Implemented in D2-T05**

```
POST /auth/mfa/enroll  (authenticated)
  → generate base32 TOTP secret
  → AES-256-GCM encrypt secret
  → INSERT np_mfa_devices (device_type='totp', totp_secret_enc, verified_at=NULL)
  → generate 8 backup codes; hash each; AES-256-GCM encrypt array
  → INSERT np_mfa_devices (device_type='backup_codes', backup_codes_enc)
  ← 200 { qr_code_url, secret_b32, backup_codes_plaintext }

POST /auth/mfa/verify  (during signin challenge)
  → lookup active TOTP device for user (disabled_at IS NULL, verified_at IS NOT NULL)
  → verify submitted TOTP code against decrypted secret (RFC 6238, ±30s window)
  → rate-limit: 5 failures → 5-min lockout
  → on first verify after enrollment: set verified_at = NOW()
  ← 200 (session elevated to MFA-verified)

POST /auth/mfa/disable  (authenticated, re-auth required)
  → verify current password or valid TOTP
  → UPDATE np_mfa_devices SET disabled_at = NOW() WHERE user_id = $1
  ← 200 { message: "MFA disabled" }
```

---

## OAuth Social Login (CSRF Protected — Amendment 9 SEC-2)

> Status: **Implemented in D2-T12**

```
GET /auth/oauth/google   (or /auth/oauth/github)
  → generate state = 32 random bytes
  → state_hash = SHA256(state), hex-encoded
  → INSERT np_oauth_states (state_hash, provider, expires_at = now + 10 min)
  → redirect to provider with state param

GET /auth/callback/:provider?code=...&state=...
  → recompute state_hash = SHA256(state)
  → SELECT from np_oauth_states WHERE state_hash = $1 AND expires_at > now
  → 400 "CSRF validation failed" if not found or expired
  → exchange code for OAuth tokens
  → upsert np_users (email from provider, email_verified = TRUE for verified providers)
  → issue session + JWT (same flow as signin)
  ← redirect to AUTH_MAGIC_LINK_BASE_URL (allowlisted paths only)
```

**redirect_uri allowlist:** only `api.nself.org/auth/callback/*` paths. No wildcards.

---

## JWT Key Rotation

1. Generate new RSA 2048-bit keypair: `openssl genrsa -out new-key.pem 2048`
2. Extract public key: `openssl rsa -in new-key.pem -pubout -out new-pub.pem`
3. Bump `AUTH_JWT_KEY_ID` in `.env.prod` (e.g., `prod-key-YYYY-MM-DD`)
4. Update `AUTH_JWT_PRIVATE_KEY` + `AUTH_JWT_PUBLIC_KEY` in `.env.secrets`
5. Add new public key to JWKS endpoint alongside the old key (24h grace period)
6. After 24h: remove old key from JWKS
7. Run `nself build && nself deploy prod`

The JWKS endpoint is at `https://auth.nself.org/.well-known/jwks.json`. Hasura is configured with `{"type":"RS256","jwk_url":"https://auth.nself.org/.well-known/jwks.json"}` so it auto-rotates when the JWKS updates.

---

## Session Revocation (Amendment 9 SEC-1)

Sessions are stored in `np_sessions` with a `revoked_at` column. The auth middleware checks `revoked_at IS NULL` on every authenticated request — this means even a valid unexpired JWT will return 401 if the session was server-side revoked.

**Revocation triggers:**
- Password reset: all sessions for the user
- Sign-out-all: all sessions for the user
- Admin revocation: specific session by ID

---

## Hasura JWT Claims

JWT payload includes Hasura claims in the namespace `https://hasura.io/jwt/claims`:

```json
{
  "https://hasura.io/jwt/claims": {
    "x-hasura-user-id": "<np_users.id>",
    "x-hasura-default-role": "nself_user",
    "x-hasura-allowed-roles": ["nself_user", "nself_admin"],
    "x-hasura-source-account-id": "<source_account_id>",
    "x-hasura-tenant-id": "<tenant_id or null>"
  }
}
```

Hasura row filters use `X-Hasura-User-Id` for `np_users` and `np_sessions`. Cloud tables also filter by `X-Hasura-Tenant-Id`.

---

## See Also

- [plugin-auth.md](plugin-auth.md) — plugin install instructions
- [Config-Auth.md](Config-Auth.md) — env var reference
- `web/backend/migrations/025_np_auth_plugin_schema.sql` — schema source
- `web/backend/plugins/auth/plugin.yaml` — plugin config
