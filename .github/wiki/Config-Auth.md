# Config, Auth

The ɳSelf auth service is a containerised instance of [nhost/hasura-auth](https://github.com/nhost/hasura-auth). It handles user registration and sign-in, issues JWTs, and provides OAuth integration for third-party providers. The service runs internally on port **4000** and is exposed through Nginx at `auth.<BASE_DOMAIN>`.

---

## Overview

```
nSelf stack
  └── auth (hasura-auth)
        ├── listens on 127.0.0.1:4000 (internal)
        ├── exposed via Nginx at https://auth.<BASE_DOMAIN>
        ├── issues JWTs verified by Hasura
        └── emails via SMTP (Mailpit in dev, real SMTP in production)
```

The auth service shares a JWT secret with Hasura. Both services read it from `HASURA_JWT_KEY`. The `AUTH_JWT_SECRET` variable is an alias that resolves to the same value, you only need to set `HASURA_JWT_KEY` once.

---

## Core Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `AUTH_VERSION` | string | `0.36.0` | Docker image tag for hasura-auth |
| `AUTH_PORT` | int | `4000` | Internal service port , do not change unless you have a conflict |
| `AUTH_CLIENT_URL` | string | `http://localhost:3000` | OAuth redirect base URL , set this to your frontend's origin |
| `AUTH_ROUTE` | string | `auth.${BASE_DOMAIN}` | Nginx subdomain for the auth service |
| `AUTH_EXTRA_REDIRECT_URLS` | string | *(empty)* | Additional OAuth redirect URLs, comma-separated |
| `AUTH_WEBAUTHN_ENABLED` | bool | `false` | Enable WebAuthn / passkey support |
| `AUTH_LOG_LEVEL` | string | `info` | Log verbosity: `debug`, `info`, `warn`, or `error` |
| `AUTH_RATE_LIMIT` | string | `30r/m` | Nginx rate limit applied to all auth endpoints |
| `AUTH_MEM_LIMIT` | string | `256m` | Container memory limit |
| `AUTH_CPU_LIMIT` | string | `0.25` | Container CPU limit (fractional cores) |

### AUTH_CLIENT_URL and redirects

`AUTH_CLIENT_URL` is the base URL that the auth service redirects users back to after OAuth sign-in. Set this to the origin of your frontend application:

```bash
# .env.dev
AUTH_CLIENT_URL=http://localhost:3000

# .env.prod
AUTH_CLIENT_URL=https://app.yourdomain.com
```

If your frontend can be reached from multiple origins (for example, a Vercel preview URL alongside your production domain), add the extra origins to `AUTH_EXTRA_REDIRECT_URLS`:

```bash
AUTH_EXTRA_REDIRECT_URLS=https://preview.yourdomain.com,https://staging.yourdomain.com
```

---

## JWT Expiry Settings

The auth service has two sets of expiry variables. Both control the same TTLs, use whichever format your frontend SDK expects:

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `AUTH_ACCESS_TOKEN_EXPIRES_IN` | int (seconds) | `900` | 15 minutes , used by the auth container |
| `AUTH_REFRESH_TOKEN_EXPIRES_IN` | int (seconds) | `2592000` | 30 days , used by the auth container |
| `AUTH_ACCESS_TOKEN_EXPIRY` | string | `15m` | Human-readable alias , convenience reference |
| `AUTH_REFRESH_TOKEN_EXPIRY` | string | `7d` | Human-readable alias , convenience reference |
| `AUTH_JWT_SECRET` | string | `${HASURA_JWT_KEY}` | Alias , resolves to `HASURA_JWT_KEY` automatically |

The integer-second variables (`AUTH_ACCESS_TOKEN_EXPIRES_IN`, `AUTH_REFRESH_TOKEN_EXPIRES_IN`) are the ones passed to the auth container. The human-readable variables (`AUTH_ACCESS_TOKEN_EXPIRY`, `AUTH_REFRESH_TOKEN_EXPIRY`) exist for documentation and tooling convenience, they are not read by the container directly.

**Example: tighter token lifetime for a high-security project**

```bash
# .env.prod
AUTH_ACCESS_TOKEN_EXPIRES_IN=300       # 5 minutes
AUTH_REFRESH_TOKEN_EXPIRES_IN=604800   # 7 days
```

---

## SMTP Configuration

The auth service sends transactional emails for magic links, email verification, and password resets. In development, ɳSelf routes all outgoing email through [Mailpit](https://mailpit.axllent.org/) (a local inbox that captures email without delivering it). In production, you point the service at a real SMTP provider.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `AUTH_SMTP_HOST` | string | `mailpit` | SMTP server hostname |
| `AUTH_SMTP_PORT` | int | `1025` | SMTP port (Mailpit uses 1025; standard SMTP uses 587 or 465) |
| `AUTH_SMTP_USER` | string | *(empty)* | SMTP username , required by most production providers |
| `AUTH_SMTP_PASS` | string | *(empty)* | SMTP password , store in `.env.secrets`, never `.env.dev` |
| `AUTH_SMTP_SECURE` | bool | `false` | Enable TLS , set to `true` when using port 465 |
| `AUTH_SMTP_SENDER` | string | `noreply@{BASE_DOMAIN}` | From address on all auth emails |
| `AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED` | bool | `false` | Require email verification before sign-in is allowed |

**Development defaults** (Mailpit, no credentials needed):

```bash
# .env.dev — these are the defaults; you only need to set them if you change them
AUTH_SMTP_HOST=mailpit
AUTH_SMTP_PORT=1025
AUTH_SMTP_SECURE=false
```

**Production example (SMTP with TLS on port 587)**:

```bash
# .env.prod
AUTH_SMTP_HOST=smtp.your-provider.com
AUTH_SMTP_PORT=587
AUTH_SMTP_SECURE=false
AUTH_SMTP_SENDER=noreply@yourdomain.com
AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED=true

# .env.secrets — credentials only, never commit this file
AUTH_SMTP_USER=apikey
AUTH_SMTP_PASS=SG.xxxxxxxxxxxxxxxxxxxxxxxxx
```

**Production example (SMTP with TLS on port 465)**:

```bash
# .env.prod
AUTH_SMTP_HOST=smtp.your-provider.com
AUTH_SMTP_PORT=465
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
```

---

## OAuth Provider Configuration

OAuth provider credentials follow a consistent naming pattern:

```
AUTH_PROVIDER_{PROVIDER}_CLIENT_ID
AUTH_PROVIDER_{PROVIDER}_CLIENT_SECRET
```

All `AUTH_PROVIDER_*` variables are passed through automatically to the auth container. You do not need to enable providers explicitly, the auth service activates a provider as soon as both its `CLIENT_ID` and `CLIENT_SECRET` (or equivalent credentials) are set.

### Supported Providers

| Provider | Variables |
|----------|-----------|
| **Google** | `AUTH_PROVIDER_GOOGLE_CLIENT_ID`, `AUTH_PROVIDER_GOOGLE_CLIENT_SECRET` |
| **GitHub** | `AUTH_PROVIDER_GITHUB_CLIENT_ID`, `AUTH_PROVIDER_GITHUB_CLIENT_SECRET` |
| **Apple** | `AUTH_PROVIDER_APPLE_CLIENT_ID`, `AUTH_PROVIDER_APPLE_TEAM_ID`, `AUTH_PROVIDER_APPLE_KEY_ID` |
| **Facebook** | `AUTH_PROVIDER_FACEBOOK_CLIENT_ID`, `AUTH_PROVIDER_FACEBOOK_CLIENT_SECRET` |
| **Twitter / X** | `AUTH_PROVIDER_TWITTER_CONSUMER_KEY`, `AUTH_PROVIDER_TWITTER_CONSUMER_SECRET` |
| **LinkedIn** | `AUTH_PROVIDER_LINKEDIN_CLIENT_ID`, `AUTH_PROVIDER_LINKEDIN_CLIENT_SECRET` |
| **Discord** | `AUTH_PROVIDER_DISCORD_CLIENT_ID`, `AUTH_PROVIDER_DISCORD_CLIENT_SECRET` |
| **Twitch** | `AUTH_PROVIDER_TWITCH_CLIENT_ID`, `AUTH_PROVIDER_TWITCH_CLIENT_SECRET` |
| **Spotify** | `AUTH_PROVIDER_SPOTIFY_CLIENT_ID`, `AUTH_PROVIDER_SPOTIFY_CLIENT_SECRET` |

Additional providers that follow the same `AUTH_PROVIDER_{PROVIDER}_*` pattern are also accepted by the auth container. Refer to the [hasura-auth provider documentation](https://github.com/nhost/hasura-auth) for the full list.

### Setting Up GitHub Login

1. Create an OAuth App at [github.com/settings/developers](https://github.com/settings/developers)
2. Set the **Authorization callback URL** to `https://auth.<BASE_DOMAIN>/signin/provider/github/callback`
3. Add the credentials to `.env.secrets`:

```bash
# .env.secrets
AUTH_PROVIDER_GITHUB_CLIENT_ID=Iv1.xxxxxxxxxxxxxxxx
AUTH_PROVIDER_GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

---

## Forwarding Any AUTH_* / HASURA_AUTH_* Engine Var

Beyond `AUTH_PROVIDER_*`, **every other variable in hasura-auth's own `AUTH_*` and `HASURA_AUTH_*` namespaces set in any `.env` file is forwarded verbatim into the auth container** at `nself build` time, additive only, so it never overrides the curated values above (JWT secret, SMTP config, allowed redirect URLs, etc.). This covers hasura-auth engine flags ɳSelf does not wrap in its own typed setting, for example:

```bash
# .env.dev
AUTH_ANONYMOUS_USERS_ENABLED=true
AUTH_DISABLE_NEW_USERS=false
AUTH_REQUIRE_EMAIL_VERIFICATION=true
HASURA_AUTH_SMTP_HOST=smtp.example.com
```

Run `nself build` (or `nself restart`) after adding one of these for the regenerated compose to pick it up. See [[Config-Hasura]] for the equivalent `HASURA_GRAPHQL_*` forwarding rule.

### Setting Up Google Login

1. Create OAuth 2.0 credentials in the [Google Cloud Console](https://console.cloud.google.com/)
2. Add `https://auth.<BASE_DOMAIN>/signin/provider/google/callback` as an authorised redirect URI
3. Add the credentials to `.env.secrets`:

```bash
# .env.secrets
AUTH_PROVIDER_GOOGLE_CLIENT_ID=xxxxxxxxxx.apps.googleusercontent.com
AUTH_PROVIDER_GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxxxxxxx
```

### Setting Up Apple Login

Apple Sign-In requires three values instead of the standard client ID / secret pair:

```bash
# .env.secrets
AUTH_PROVIDER_APPLE_CLIENT_ID=com.yourapp.service    # Apple Services ID
AUTH_PROVIDER_APPLE_TEAM_ID=XXXXXXXXXX               # 10-character Team ID
AUTH_PROVIDER_APPLE_KEY_ID=XXXXXXXXXX                # 10-character Key ID
```

The Apple private key (`.p8` file) is loaded separately, refer to the [hasura-auth Apple configuration guide](https://github.com/nhost/hasura-auth) for how to mount the key file into the container.

---

## Production Checklist

Before going live, work through the following:

**SMTP**
- Replace `AUTH_SMTP_HOST=mailpit` with your real SMTP provider host
- Set `AUTH_SMTP_PORT`, `AUTH_SMTP_SECURE`, `AUTH_SMTP_USER`, and `AUTH_SMTP_PASS` in `.env.secrets`
- Verify `AUTH_SMTP_SENDER` matches a verified sender domain on your email provider
- Consider enabling `AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED=true` to block unverified sign-ins

**Rate limiting**
- The default `AUTH_RATE_LIMIT=30r/m` is reasonable for most applications. Tighten it if you see abuse:

```bash
# .env.prod
AUTH_RATE_LIMIT=10r/m   # stricter — 10 requests per minute per IP
```

**Frontend redirect URL**
- Set `AUTH_CLIENT_URL` to your production frontend origin, not `localhost`
- List any additional allowed origins in `AUTH_EXTRA_REDIRECT_URLS`

**Token lifetimes**
- The default 15-minute access token is suitable for most apps
- Shorten `AUTH_ACCESS_TOKEN_EXPIRES_IN` for high-security contexts
- The default 30-day refresh token is generous, adjust to match your session policy

**Resource limits**
- The default `AUTH_MEM_LIMIT=256m` and `AUTH_CPU_LIMIT=0.25` are conservative. Increase them if you have a high sign-in volume

**WebAuthn / passkeys**
- Set `AUTH_WEBAUTHN_ENABLED=true` if you want to support hardware security keys and passkeys (requires a valid HTTPS origin, will not work on `http://localhost`)

---

← [[Config-Hasura]] | [[Configuration]] | [[Config-Nginx]] →
