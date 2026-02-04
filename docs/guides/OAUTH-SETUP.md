# OAuth Setup Guide

Complete guide for setting up OAuth authentication in nself.

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Provider Configuration](#provider-configuration)
4. [Advanced Configuration](#advanced-configuration)
5. [Security Best Practices](#security-best-practices)
6. [Troubleshooting](#troubleshooting)

---

## Overview

nself provides pre-built OAuth handlers for popular authentication providers:

- **Google OAuth 2.0** - Sign in with Google
- **GitHub OAuth 2.0** - Sign in with GitHub
- **Microsoft OAuth 2.0** - Sign in with Microsoft / Azure AD
- **Slack OAuth 2.0** - Sign in with Slack

### Architecture

```
┌─────────────┐      ┌──────────────────┐      ┌────────────┐
│   Frontend  │─────▶│ OAuth Handlers   │─────▶│  Provider  │
│             │◀─────│    Service       │◀─────│  (Google)  │
└─────────────┘      └──────────────────┘      └────────────┘
                             │
                             ▼
                     ┌──────────────┐
                     │ Nhost Auth   │
                     │  + Hasura    │
                     └──────────────┘
```

**Flow:**
1. User clicks "Sign in with Google" on frontend
2. Frontend redirects to OAuth Handlers service
3. OAuth Handlers redirects to provider (Google)
4. User authorizes on provider's site
5. Provider redirects back to OAuth Handlers with code
6. OAuth Handlers exchanges code for tokens
7. OAuth Handlers gets user profile
8. OAuth Handlers creates/updates user in Nhost
9. OAuth Handlers generates JWT
10. Frontend receives JWT and authenticates user

---

## Quick Start

### 1. Install OAuth Handlers Service

```bash
nself oauth install
```

This copies the OAuth handlers template to `services/oauth-handlers/`.

### 2. Enable Providers

```bash
# Enable one or more providers
nself oauth enable --providers google,github,slack
```

### 3. Configure Credentials

Each provider requires OAuth app credentials. See [Provider Configuration](#provider-configuration) for detailed setup instructions.

#### Google Example:

```bash
nself oauth config google \
  --client-id=123456789.apps.googleusercontent.com \
  --client-secret=GOCSPX-xxxxx
```

#### GitHub Example:

```bash
nself oauth config github \
  --client-id=Iv1.abcdef123456 \
  --client-secret=ghp_xxxxx
```

### 4. Rebuild and Start

```bash
nself build
nself start
```

### 5. Test OAuth Flow

Visit: `http://localhost:3100/oauth/google`

You'll be redirected to Google's login page, then back to your frontend with a JWT token.

---

## Provider Configuration

### Google OAuth 2.0

#### 1. Create OAuth App

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Navigate to **APIs & Services** > **Credentials**
4. Click **Create Credentials** > **OAuth client ID**
5. Application type: **Web application**
6. Authorized redirect URIs:
   - Development: `http://localhost:3100/oauth/google/callback`
   - Production: `https://your-domain.com/oauth/google/callback`

#### 2. Configure in nself

```bash
nself oauth enable --providers google

nself oauth config google \
  --client-id=YOUR_CLIENT_ID.apps.googleusercontent.com \
  --client-secret=GOCSPX-YOUR_CLIENT_SECRET \
  --callback-url=http://localhost:3100/oauth/google/callback
```

#### 3. Test

```bash
nself oauth test google
```

---

### GitHub OAuth 2.0

#### 1. Create OAuth App

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click **New OAuth App**
3. Fill in:
   - Application name: Your app name
   - Homepage URL: `http://localhost:3000`
   - Authorization callback URL: `http://localhost:3100/oauth/github/callback`

#### 2. Configure in nself

```bash
nself oauth enable --providers github

nself oauth config github \
  --client-id=Iv1.YOUR_CLIENT_ID \
  --client-secret=YOUR_CLIENT_SECRET \
  --callback-url=http://localhost:3100/oauth/github/callback
```

#### 3. Test

```bash
nself oauth test github
```

---

### Microsoft OAuth 2.0

#### 1. Register App in Azure AD

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to **Azure Active Directory** > **App registrations**
3. Click **New registration**
4. Fill in:
   - Name: Your app name
   - Supported account types: Choose appropriate option
   - Redirect URI: `http://localhost:3100/oauth/microsoft/callback`
5. After creation, go to **Certificates & secrets**
6. Create a new client secret

#### 2. Configure in nself

```bash
nself oauth enable --providers microsoft

nself oauth config microsoft \
  --client-id=YOUR_APPLICATION_ID \
  --client-secret=YOUR_CLIENT_SECRET \
  --tenant-id=YOUR_TENANT_ID \
  --callback-url=http://localhost:3100/oauth/microsoft/callback
```

**Note:** Use `tenant-id=common` for multi-tenant apps.

#### 3. Test

```bash
nself oauth test microsoft
```

---

### Slack OAuth 2.0

#### 1. Create Slack App

1. Go to [Slack API](https://api.slack.com/apps)
2. Click **Create New App** > **From scratch**
3. Fill in app name and workspace
4. Navigate to **OAuth & Permissions**
5. Add redirect URL: `http://localhost:3100/oauth/slack/callback`
6. Add OAuth scopes:
   - `openid`
   - `profile`
   - `email`

#### 2. Configure in nself

```bash
nself oauth enable --providers slack

nself oauth config slack \
  --client-id=YOUR_CLIENT_ID.apps.slack.com \
  --client-secret=YOUR_CLIENT_SECRET \
  --callback-url=http://localhost:3100/oauth/slack/callback
```

#### 3. Test

```bash
nself oauth test slack
```

---

## Advanced Configuration

### Custom Callback URLs

By default, callback URLs are generated as:
```
http://{BASE_DOMAIN}:{PORT}/oauth/{provider}/callback
```

To use custom callback URLs:

```bash
nself oauth config google \
  --client-id=xxx \
  --client-secret=xxx \
  --callback-url=https://auth.yourdomain.com/oauth/google/callback
```

### Environment-Specific Configuration

#### Development (.env.dev)
```env
OAUTH_GOOGLE_ENABLED=true
OAUTH_GOOGLE_CLIENT_ID=dev-client-id
OAUTH_GOOGLE_CLIENT_SECRET=dev-secret
OAUTH_GOOGLE_CALLBACK_URL=http://localhost:3100/oauth/google/callback
```

#### Production (.env.prod)
```env
OAUTH_GOOGLE_ENABLED=true
OAUTH_GOOGLE_CLIENT_ID=prod-client-id
OAUTH_GOOGLE_CLIENT_SECRET=prod-secret
OAUTH_GOOGLE_CALLBACK_URL=https://yourdomain.com/oauth/google/callback
```

### Multiple Tenants (Microsoft)

For Azure AD multi-tenant apps:

```bash
nself oauth config microsoft \
  --client-id=xxx \
  --client-secret=xxx \
  --tenant-id=common
```

For single tenant:

```bash
nself oauth config microsoft \
  --client-id=xxx \
  --client-secret=xxx \
  --tenant-id=your-tenant-uuid
```

---

## Security Best Practices

### 1. Use HTTPS in Production

**Never** use HTTP in production:

```env
# ❌ Bad - HTTP in production
OAUTH_GOOGLE_CALLBACK_URL=http://yourdomain.com/oauth/google/callback

# ✅ Good - HTTPS in production
OAUTH_GOOGLE_CALLBACK_URL=https://yourdomain.com/oauth/google/callback
```

### 2. Rotate Secrets Regularly

```bash
# Update client secret
nself oauth config google \
  --client-id=existing-id \
  --client-secret=new-secret
```

### 3. Restrict Callback URLs

Only add necessary callback URLs to your OAuth app configuration:

- Development: `http://localhost:3100/oauth/{provider}/callback`
- Staging: `https://staging.yourdomain.com/oauth/{provider}/callback`
- Production: `https://yourdomain.com/oauth/{provider}/callback`

### 4. Use Environment Variables

Store credentials in `.env.local` (gitignored) or `.secrets`:

```env
# .env.local
OAUTH_GOOGLE_CLIENT_SECRET=your-secret-here
```

**Never commit secrets to git!**

### 5. Enable CORS Properly

The OAuth handlers service only allows requests from your frontend:

```env
FRONTEND_URL=http://localhost:3000
```

In production:

```env
FRONTEND_URL=https://yourdomain.com
```

### 6. JWT Security

Use strong JWT secrets:

```env
JWT_SECRET=$(openssl rand -hex 32)
JWT_EXPIRES_IN=7d
```

---

## Troubleshooting

### Common Issues

#### 1. "Provider not enabled" Error

**Cause:** Provider is not enabled in `.env` file.

**Solution:**
```bash
nself oauth enable --providers google
```

#### 2. "Invalid state" Error

**Cause:** State parameter validation failed (CSRF protection).

**Possible Reasons:**
- State expired (>10 minutes old)
- Tampering attempt
- Session/cookie issues

**Solution:**
- Retry the OAuth flow
- Check browser cookies are enabled
- Ensure OAuth handlers service is running

#### 3. "Missing client credentials" Error

**Cause:** Client ID or secret not configured.

**Solution:**
```bash
nself oauth config google \
  --client-id=xxx \
  --client-secret=xxx
```

#### 4. "Redirect URI mismatch" Error

**Cause:** Callback URL in `.env` doesn't match OAuth app configuration.

**Solution:**
1. Check callback URL in `.env`:
   ```bash
   grep OAUTH_GOOGLE_CALLBACK_URL .env.dev
   ```
2. Ensure it matches the URL in Google Cloud Console
3. Update if needed:
   ```bash
   nself oauth config google \
     --client-id=xxx \
     --client-secret=xxx \
     --callback-url=http://localhost:3100/oauth/google/callback
   ```

#### 5. OAuth Handlers Service Not Running

**Check status:**
```bash
nself oauth status
docker ps | grep oauth-handlers
```

**Start service:**
```bash
nself start
```

**View logs:**
```bash
docker logs oauth-handlers
```

### Testing OAuth Flow

#### 1. List Enabled Providers

```bash
nself oauth list
```

#### 2. Test Provider Configuration

```bash
nself oauth test google
```

#### 3. Check Service Health

```bash
curl http://localhost:3100/health
```

Expected response:
```json
{
  "status": "healthy",
  "service": "oauth-handlers",
  "timestamp": "2026-01-30T12:00:00.000Z",
  "providers": ["google", "github"]
}
```

#### 4. List Available Providers

```bash
curl http://localhost:3100/oauth/providers
```

#### 5. Initiate OAuth Flow

Visit in browser:
```
http://localhost:3100/oauth/google
```

Or with redirect:
```
http://localhost:3100/oauth/google?redirect=/dashboard
```

---

## Frontend Integration

See [Frontend Integration Guide](./OAUTH-FRONTEND-INTEGRATION.md) for detailed frontend setup.

Quick example:

```typescript
// React/Next.js example
function LoginButton() {
  const handleGoogleLogin = () => {
    window.location.href = 'http://localhost:3100/oauth/google?redirect=/dashboard';
  };

  return (
    <button onClick={handleGoogleLogin}>
      Sign in with Google
    </button>
  );
}

// Handle callback
useEffect(() => {
  const params = new URLSearchParams(window.location.search);
  const token = params.get('token');

  if (token) {
    localStorage.setItem('authToken', token);
    window.location.href = '/dashboard';
  }
}, []);
```

---

## Next Steps

- [Frontend Integration Guide](./OAUTH-FRONTEND-INTEGRATION.md)
- [OAuth API Reference](../reference/api/OAUTH-API.md)
- [OAuth CLI Reference](../commands/oauth.md)
- [Security Best Practices](./OAUTH-SECURITY.md)

---

**Updated:** January 30, 2026
**Version:** nself v0.8.0+
