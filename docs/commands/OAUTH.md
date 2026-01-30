# nself oauth - OAuth Provider Management

**Version**: 0.7.0+
**Status**: ✅ Production Ready
**Category**: Authentication & OAuth

---

## Overview

The `nself oauth` command provides comprehensive OAuth 2.0 and OpenID Connect (OIDC) provider management for your nself deployment. Support for 15+ providers enables social login, enterprise SSO, and third-party integrations.

**Why Use OAuth?**
- Enable social login (Google, GitHub, Facebook, etc.)
- Enterprise SSO with Microsoft, custom OIDC
- Third-party integrations (Slack, Discord, Spotify, etc.)
- Passwordless authentication
- Better user experience (no password to remember)

**Supported Providers (15):**
Google, GitHub, Microsoft, Facebook, Apple, Twitter/X, LinkedIn, Slack, Discord, Twitch, GitLab, Bitbucket, Spotify, Custom OIDC, Generic OAuth2

---

## Quick Start

### 1. Install OAuth Handlers Service

```bash
# Install the OAuth handlers service (one-time setup)
nself oauth install
```

**What This Does:**
- Creates `services/oauth-handlers/` directory
- Copies OAuth provider implementations
- Configures Express.js server for OAuth callbacks
- Sets up database tables for OAuth accounts

### 2. Enable Providers

```bash
# Enable multiple providers at once
nself oauth enable --providers google,github,slack

# Or enable individually
nself oauth enable --providers google
nself oauth enable --providers github
```

### 3. Configure Provider Credentials

```bash
# Configure Google OAuth
nself oauth config google \
  --client-id=123456789.apps.googleusercontent.com \
  --client-secret=GOCSPX-your-secret-here

# Configure GitHub OAuth
nself oauth config github \
  --client-id=Iv1.abc123def456 \
  --client-secret=your-github-secret-here
```

### 4. Build and Start

```bash
# Rebuild with OAuth configuration
nself build

# Start services
nself start
```

### 5. Test OAuth Flow

```bash
# Test provider configuration
nself oauth test google

# Output shows:
# - Provider status (enabled/disabled)
# - Credentials configured (yes/no)
# - Callback URL
# - Test authentication URL
```

---

## Command Reference

### `nself oauth install`

Install OAuth handlers service (one-time setup).

**Syntax:**
```bash
nself oauth install
```

**What Gets Installed:**

```
services/oauth-handlers/
├── Dockerfile                    # Node.js OAuth server
├── package.json                  # Dependencies
├── src/
│   ├── index.js                  # Express server
│   ├── routes/
│   │   ├── google.js             # Google OAuth routes
│   │   ├── github.js             # GitHub OAuth routes
│   │   ├── slack.js              # Slack OAuth routes
│   │   └── ... (other providers)
│   ├── callbacks/
│   │   └── oauth-callback.js     # OAuth callback handler
│   └── utils/
│       ├── token-refresh.js      # Auto token refresh
│       └── account-linking.js    # Link accounts
└── .env.oauth                    # OAuth configuration
```

**Example Output:**
```
ℹ Installing OAuth handlers service...
ℹ Copying OAuth handlers template...
ℹ Processing template files...
ℹ Configuring service...
✓ OAuth handlers service installed at: services/oauth-handlers

ℹ Next steps:
  1. Configure OAuth providers: nself oauth enable --providers google,github
  2. Set credentials: nself oauth config google --client-id=xxx --client-secret=xxx
  3. Rebuild services: nself build
  4. Start services: nself start
```

---

### `nself oauth enable`

Enable OAuth providers.

**Syntax:**
```bash
nself oauth enable --providers=<provider1,provider2,...>
```

**Options:**
- `--providers=<list>` - Comma-separated list of providers (required)

**Supported Providers:**
- `google` - Google OAuth (most common)
- `github` - GitHub OAuth (developer accounts)
- `microsoft` - Microsoft/Azure AD (enterprise)
- `facebook` - Facebook Login
- `apple` - Sign in with Apple
- `twitter` - Twitter/X OAuth
- `linkedin` - LinkedIn OAuth
- `slack` - Slack workspace integration
- `discord` - Discord OAuth
- `twitch` - Twitch OAuth
- `gitlab` - GitLab OAuth
- `bitbucket` - Bitbucket OAuth
- `spotify` - Spotify OAuth
- `oidc` - Custom OpenID Connect provider
- `oauth2` - Generic OAuth 2.0

**Examples:**
```bash
# Enable single provider
nself oauth enable --providers google

# Enable multiple providers
nself oauth enable --providers google,github,slack

# Enable all social providers
nself oauth enable --providers google,github,facebook,twitter,linkedin

# Enable enterprise providers
nself oauth enable --providers microsoft,oidc
```

**Output:**
```
✓ Enabled google OAuth
✓ Enabled github OAuth
✓ Enabled slack OAuth

ℹ Next step: Configure credentials with 'nself oauth config <provider>'
```

---

### `nself oauth disable`

Disable OAuth providers.

**Syntax:**
```bash
nself oauth disable --providers=<provider1,provider2,...>
```

**Options:**
- `--providers=<list>` - Comma-separated list of providers (required)

**Examples:**
```bash
# Disable single provider
nself oauth disable --providers google

# Disable multiple providers
nself oauth disable --providers google,github
```

**Output:**
```
✓ Disabled google OAuth
✓ Disabled github OAuth

⚠ Run 'nself build' to apply changes
```

---

### `nself oauth config`

Configure OAuth provider credentials.

**Syntax:**
```bash
nself oauth config <provider> [options]
```

**Options:**
- `<provider>` - Provider name (required)
- `--client-id=<id>` - OAuth client ID (required)
- `--client-secret=<secret>` - OAuth client secret (required)
- `--tenant-id=<id>` - Tenant ID (Microsoft only)
- `--callback-url=<url>` - Custom callback URL (optional)

**Provider-Specific Configuration:**

#### Google OAuth
```bash
nself oauth config google \
  --client-id=123456789.apps.googleusercontent.com \
  --client-secret=GOCSPX-your-secret-here
```

**Where to Get Credentials:**
1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create project or select existing
3. Navigate to "APIs & Services" → "Credentials"
4. Create "OAuth 2.0 Client ID"
5. Add authorized redirect URI: `https://auth.yourdomain.com/oauth/google/callback`

#### GitHub OAuth
```bash
nself oauth config github \
  --client-id=Iv1.abc123def456 \
  --client-secret=your-github-secret-here
```

**Where to Get Credentials:**
1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Set callback URL: `https://auth.yourdomain.com/oauth/github/callback`
4. Copy Client ID and Client Secret

#### Microsoft OAuth (Azure AD)
```bash
nself oauth config microsoft \
  --client-id=abc123-def456-ghi789 \
  --client-secret=your-secret-here \
  --tenant-id=your-tenant-id
```

**Where to Get Credentials:**
1. Go to [Azure Portal](https://portal.azure.com)
2. Navigate to "Azure Active Directory" → "App registrations"
3. Register new application
4. Add redirect URI: `https://auth.yourdomain.com/oauth/microsoft/callback`
5. Copy Application (client) ID and Directory (tenant) ID
6. Create client secret under "Certificates & secrets"

#### Facebook Login
```bash
nself oauth config facebook \
  --client-id=123456789012345 \
  --client-secret=your-facebook-secret
```

**Where to Get Credentials:**
1. Go to [Facebook Developers](https://developers.facebook.com)
2. Create App → Consumer
3. Add "Facebook Login" product
4. Settings → Valid OAuth Redirect URIs: `https://auth.yourdomain.com/oauth/facebook/callback`
5. Copy App ID and App Secret

#### Sign in with Apple
```bash
nself oauth config apple \
  --client-id=com.yourdomain.auth \
  --client-secret=<generated-secret>
```

**Where to Get Credentials:**
1. Go to [Apple Developer Portal](https://developer.apple.com)
2. "Certificates, IDs & Profiles" → "Identifiers"
3. Create App ID with "Sign in with Apple" capability
4. Create Service ID for web authentication
5. Configure Return URLs: `https://auth.yourdomain.com/oauth/apple/callback`

#### Slack Workspace Integration
```bash
nself oauth config slack \
  --client-id=123456789.123456789 \
  --client-secret=your-slack-secret
```

**Where to Get Credentials:**
1. Go to [Slack API Dashboard](https://api.slack.com/apps)
2. Create New App → From scratch
3. OAuth & Permissions → Redirect URLs: `https://auth.yourdomain.com/oauth/slack/callback`
4. Copy Client ID and Client Secret

#### Custom OpenID Connect (Enterprise SSO)
```bash
nself oauth config oidc \
  --client-id=your-client-id \
  --client-secret=your-client-secret \
  --issuer-url=https://sso.yourcompany.com
```

**For Auth0:**
```bash
nself oauth config oidc \
  --client-id=your-auth0-client-id \
  --client-secret=your-auth0-secret \
  --issuer-url=https://your-tenant.auth0.com
```

**For Okta:**
```bash
nself oauth config oidc \
  --client-id=your-okta-client-id \
  --client-secret=your-okta-secret \
  --issuer-url=https://your-domain.okta.com
```

---

### `nself oauth test`

Test OAuth provider configuration.

**Syntax:**
```bash
nself oauth test <provider>
```

**Example:**
```bash
nself oauth test google
```

**Output:**
```
Testing Google OAuth Configuration
=====================================

Provider Status: ENABLED
Credentials: CONFIGURED
Callback URL: https://auth.yourdomain.com/oauth/google/callback

Environment Variables:
  OAUTH_GOOGLE_ENABLED=true
  OAUTH_GOOGLE_CLIENT_ID=123456789.apps.googleusercontent.com
  OAUTH_GOOGLE_CLIENT_SECRET=***************GOCSPX

OAuth Flow:
  1. Authorization URL:
     https://accounts.google.com/o/oauth2/v2/auth?
       client_id=123456789.apps.googleusercontent.com
       &redirect_uri=https://auth.yourdomain.com/oauth/google/callback
       &response_type=code
       &scope=openid+profile+email

  2. User authorizes

  3. Callback to: https://auth.yourdomain.com/oauth/google/callback?code=xxx

  4. Token exchange

  5. User profile fetched

✓ Configuration looks good!

To test manually, visit:
https://auth.yourdomain.com/oauth/google/login
```

---

### `nself oauth list`

List all OAuth providers and their status.

**Syntax:**
```bash
nself oauth list [--format=table|json]
```

**Options:**
- `--format=table` - Table format (default)
- `--format=json` - JSON output

**Example:**
```bash
nself oauth list
```

**Output:**
```
OAuth Providers Status
=======================

 Provider   | Status   | Configured | Accounts
------------|----------|------------|----------
 google     | ENABLED  | YES        | 45
 github     | ENABLED  | YES        | 23
 slack      | ENABLED  | YES        | 8
 microsoft  | DISABLED | NO         | 0
 facebook   | DISABLED | NO         | 0
 apple      | DISABLED | NO         | 0
 twitter    | DISABLED | NO         | 0
 linkedin   | DISABLED | NO         | 0
 discord    | DISABLED | NO         | 0
 twitch     | DISABLED | NO         | 0
 gitlab     | DISABLED | NO         | 0
 bitbucket  | DISABLED | NO         | 0
 spotify    | DISABLED | NO         | 0
 oidc       | DISABLED | NO         | 0
 oauth2     | DISABLED | NO         | 0

Total Enabled: 3
Total Configured: 3
Total User Accounts: 76
```

**JSON Output:**
```bash
nself oauth list --format=json
```

```json
{
  "providers": [
    {
      "name": "google",
      "enabled": true,
      "configured": true,
      "accounts": 45,
      "callback_url": "https://auth.yourdomain.com/oauth/google/callback"
    },
    {
      "name": "github",
      "enabled": true,
      "configured": true,
      "accounts": 23,
      "callback_url": "https://auth.yourdomain.com/oauth/github/callback"
    }
  ],
  "summary": {
    "total_providers": 15,
    "enabled": 3,
    "configured": 3,
    "total_accounts": 76
  }
}
```

---

### `nself oauth status`

Show OAuth service status and health.

**Syntax:**
```bash
nself oauth status
```

**Output:**
```
OAuth Service Status
=====================

Service: RUNNING
Container: demo-app_oauth-handlers
Port: 3100
Health: HEALTHY

Endpoints:
  - https://oauth.yourdomain.com
  - https://auth.yourdomain.com/oauth/*

Enabled Providers: 3
  ✓ google
  ✓ github
  ✓ slack

Token Refresh Service: RUNNING
  - Last run: 2 minutes ago
  - Tokens refreshed: 12
  - Errors: 0

Database Tables:
  ✓ auth.oauth_accounts (76 records)
  ✓ auth.oauth_tokens (76 records)
  ✓ auth.oauth_refresh_log (1,234 records)

Logs:
  To view logs: nself logs oauth-handlers --tail 100
```

---

### `nself oauth accounts`

List OAuth accounts for a user.

**Syntax:**
```bash
nself oauth accounts <user_id> [--format=table|json]
```

**Options:**
- `<user_id>` - User UUID (required)
- `--format=table` - Table format (default)
- `--format=json` - JSON output

**Example:**
```bash
nself oauth accounts 123e4567-e89b-12d3-a456-426614174000
```

**Output:**
```
OAuth Accounts for User: john@example.com
===========================================

 Provider | Account ID        | Email                  | Linked At           | Last Used
----------|-------------------|------------------------|---------------------|--------------------
 google   | 109876543210      | john@gmail.com         | 2026-01-15 10:00:00 | 2026-01-30 09:30:00
 github   | johndoe           | john@example.com       | 2026-01-20 14:30:00 | 2026-01-29 16:45:00
 slack    | U01ABC123         | john@example.com       | 2026-01-25 11:15:00 | 2026-01-30 08:00:00

Total Linked Accounts: 3
```

---

### `nself oauth refresh`

Manage OAuth token refresh service.

**Syntax:**
```bash
nself oauth refresh <action>
```

**Actions:**
- `status` - Show refresh service status
- `start` - Start automatic refresh service
- `stop` - Stop automatic refresh service
- `once` - Run refresh once manually

**Token Refresh Service:**

OAuth tokens expire (typically 1 hour for access tokens). The refresh service automatically renews tokens using refresh tokens before they expire.

**Check Status:**
```bash
nself oauth refresh status
```

**Output:**
```
Token Refresh Service Status
==============================

Status: RUNNING
Mode: AUTOMATIC
Interval: Every 15 minutes
Last Run: 3 minutes ago

Statistics (Last 24 hours):
  - Refresh attempts: 96
  - Successful: 94
  - Failed: 2
  - Tokens refreshed: 342

Recent Refresh Log:
  2026-01-30 09:45:00 | google   | user-123 | SUCCESS
  2026-01-30 09:45:01 | github   | user-456 | SUCCESS
  2026-01-30 09:45:02 | google   | user-789 | FAILED (refresh_token expired)
```

**Manual Refresh:**
```bash
# Refresh all expired tokens now
nself oauth refresh once
```

**Output:**
```
Running token refresh...
✓ Refreshed 12 tokens
⚠ 1 token failed to refresh (user-789@google)
ℹ Failed tokens require user to re-authenticate
```

---

### `nself oauth link`

Link OAuth provider to user account.

**Syntax:**
```bash
nself oauth link <user_id> <provider>
```

**Options:**
- `<user_id>` - User UUID (required)
- `<provider>` - Provider name (required)

**Example:**
```bash
nself oauth link 123e4567-e89b-12d3-a456-426614174000 github
```

**Output:**
```
Linking GitHub to user account...

Authorization URL:
https://github.com/login/oauth/authorize?client_id=xxx&state=link_token_xxx

Please open this URL in browser to complete linking:
https://auth.yourdomain.com/oauth/github/link?token=link_token_xxx

Waiting for authorization... (timeout in 5 minutes)
```

**Use Case:** Admin linking a provider to a user's account programmatically.

---

### `nself oauth unlink`

Unlink OAuth provider from user account.

**Syntax:**
```bash
nself oauth unlink <user_id> <provider>
```

**Options:**
- `<user_id>` - User UUID (required)
- `<provider>` - Provider name (required)

**Example:**
```bash
nself oauth unlink 123e4567-e89b-12d3-a456-426614174000 github
```

**Output:**
```
Unlinking GitHub from user account...
✓ GitHub account unlinked successfully

Remaining linked accounts: 2 (google, slack)
```

---

## Complete OAuth Setup Workflow

### Phase 1: Install OAuth Service

```bash
# 1. Install OAuth handlers
nself oauth install

# 2. Verify installation
ls services/oauth-handlers/
```

### Phase 2: Enable and Configure Providers

```bash
# 3. Enable providers
nself oauth enable --providers google,github

# 4. Configure Google
nself oauth config google \
  --client-id=YOUR_GOOGLE_CLIENT_ID \
  --client-secret=YOUR_GOOGLE_SECRET

# 5. Configure GitHub
nself oauth config github \
  --client-id=YOUR_GITHUB_CLIENT_ID \
  --client-secret=YOUR_GITHUB_SECRET
```

### Phase 3: Build and Deploy

```bash
# 6. Rebuild with OAuth configuration
nself build

# 7. Start services
nself start

# 8. Verify OAuth service is running
nself oauth status
```

### Phase 4: Test OAuth Flow

```bash
# 9. Test provider configuration
nself oauth test google
nself oauth test github

# 10. Try OAuth login in browser
# Navigate to: https://auth.yourdomain.com/oauth/google/login
```

### Phase 5: Monitor

```bash
# 11. Check OAuth accounts
nself oauth list

# 12. View logs
nself logs oauth-handlers --tail 100

# 13. Monitor token refresh
nself oauth refresh status
```

---

## Provider Setup Guides

For detailed provider-specific setup instructions, see:

- [Google OAuth Setup](../providers/oauth/GOOGLE.md)
- [GitHub OAuth Setup](../providers/oauth/GITHUB.md)
- [Microsoft OAuth Setup](../providers/oauth/MICROSOFT.md)
- [Facebook OAuth Setup](../providers/oauth/FACEBOOK.md)
- [Apple Sign In Setup](../providers/oauth/APPLE.md)
- [Slack OAuth Setup](../providers/oauth/SLACK.md)
- [Custom OIDC Setup](../providers/oauth/CUSTOM-OIDC.md)
- [All OAuth Providers](../guides/OAUTH-COMPLETE-FLOWS.md)

---

## Client Integration

### JavaScript/TypeScript SDK

```typescript
import { Auth } from '@nself/client';

const auth = new Auth({
  url: 'https://auth.yourdomain.com'
});

// Sign in with Google
await auth.signIn({ provider: 'google' });

// Sign in with GitHub
await auth.signIn({ provider: 'github' });

// Sign in with popup
const { user, session } = await auth.signInWithOAuth({
  provider: 'google',
  mode: 'popup' // or 'redirect'
});

// Sign in with redirect (default)
auth.signInWithOAuth({ provider: 'github' });
// User will be redirected back to callback URL
```

### React Hook

```typescript
import { useAuth } from '@nself/react';

function LoginButtons() {
  const { signInWithOAuth } = useAuth();

  return (
    <>
      <button onClick={() => signInWithOAuth({ provider: 'google' })}>
        Sign in with Google
      </button>
      <button onClick={() => signInWithOAuth({ provider: 'github' })}>
        Sign in with GitHub
      </button>
      <button onClick={() => signInWithOAuth({ provider: 'microsoft' })}>
        Sign in with Microsoft
      </button>
    </>
  );
}
```

### Direct URL Access

Users can also access OAuth login directly via URL:

```
https://auth.yourdomain.com/oauth/google/login
https://auth.yourdomain.com/oauth/github/login
https://auth.yourdomain.com/oauth/slack/login
```

---

## Database Schema

### OAuth Accounts Table

```sql
CREATE TABLE auth.oauth_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  provider VARCHAR(50) NOT NULL,
  provider_user_id VARCHAR(255) NOT NULL,
  email VARCHAR(255),
  name VARCHAR(255),
  avatar_url TEXT,
  access_token TEXT,
  refresh_token TEXT,
  expires_at TIMESTAMPTZ,
  scope TEXT,
  token_type VARCHAR(50),
  id_token TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  last_used_at TIMESTAMPTZ,
  UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_oauth_accounts_user_id ON auth.oauth_accounts(user_id);
CREATE INDEX idx_oauth_accounts_provider ON auth.oauth_accounts(provider);
```

### OAuth Token Refresh Log

```sql
CREATE TABLE auth.oauth_refresh_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  oauth_account_id UUID REFERENCES auth.oauth_accounts(id) ON DELETE CASCADE,
  provider VARCHAR(50),
  user_id UUID,
  status VARCHAR(20), -- 'success', 'failed'
  error_message TEXT,
  refreshed_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Security Best Practices

### 1. Use HTTPS Only

OAuth requires HTTPS in production:

```bash
# Ensure SSL is configured
nself ssl status

# Generate SSL certificate if needed
nself ssl generate --domain yourdomain.com
```

### 2. Secure Client Secrets

Never commit secrets to git:

```bash
# Store in .env.local (gitignored)
OAUTH_GOOGLE_CLIENT_SECRET=your-secret-here

# Or use secrets management
nself secrets set OAUTH_GOOGLE_CLIENT_SECRET your-secret-here
```

### 3. Validate Redirect URIs

Always whitelist exact callback URLs in provider dashboards:

```
✅ https://auth.yourdomain.com/oauth/google/callback
❌ https://auth.yourdomain.com/*
❌ https://yourdomain.com/*
```

### 4. Implement CSRF Protection

OAuth state parameter prevents CSRF attacks (handled automatically by nself):

```javascript
// Automatic in nself OAuth handlers
const state = crypto.randomBytes(32).toString('hex');
// State verified on callback
```

### 5. Handle Token Refresh

Enable automatic token refresh:

```bash
# Start refresh service
nself oauth refresh start

# Monitor refresh logs
nself logs oauth-handlers --filter "token refresh"
```

### 6. Audit OAuth Activity

Monitor OAuth authentication:

```bash
# View OAuth audit log
nself audit oauth --days 7

# Export OAuth events
nself audit export --type oauth --format json > oauth-audit.json
```

---

## Troubleshooting

### OAuth Login Fails with "Redirect URI Mismatch"

**Problem:** Provider rejects callback URL

**Solution:**
1. Check configured callback URL in provider dashboard
2. Ensure exact match (including trailing slash)
3. Verify HTTPS in production

```bash
# Check current configuration
nself oauth test google

# Look for callback URL in output
# Example: https://auth.yourdomain.com/oauth/google/callback

# Make sure this EXACT URL is in Google Cloud Console
```

### Token Refresh Failing

**Problem:** "refresh_token expired" or "invalid_grant" errors

**Solution:**
1. User must re-authenticate
2. Check token expiration policy in provider settings
3. Some providers (Google) expire refresh tokens after 6 months of inactivity

```bash
# Check refresh status
nself oauth refresh status

# Force refresh once
nself oauth refresh once

# View failed refresh attempts
nself logs oauth-handlers --filter "refresh failed"
```

### "Client secret mismatch" Error

**Problem:** OAuth provider rejects client secret

**Solution:**
1. Verify secret is correct (no trailing spaces)
2. Re-generate secret in provider dashboard
3. Update configuration

```bash
# Update secret
nself oauth config google \
  --client-id=SAME_CLIENT_ID \
  --client-secret=NEW_SECRET

# Rebuild
nself build && nself start
```

### OAuth Service Not Starting

**Problem:** Container fails to start

**Solution:**
```bash
# Check logs
nself logs oauth-handlers

# Common issues:
# 1. Missing environment variables
nself config get | grep OAUTH

# 2. Port conflict (default 3100)
docker ps | grep 3100

# 3. Database connection
nself db status
```

### User Can't Link Multiple Providers

**Problem:** User already has account with different provider

**Solution:**

Account linking is automatic if emails match:

```bash
# Check user's linked accounts
nself oauth accounts <user_id>

# Manually link if needed
nself oauth link <user_id> github
```

---

## Performance

### Token Caching

OAuth tokens are cached in database to avoid repeated API calls:

```sql
-- Check token cache
SELECT provider, COUNT(*) as accounts,
       COUNT(CASE WHEN expires_at > NOW() THEN 1 END) as valid_tokens
FROM auth.oauth_accounts
GROUP BY provider;
```

### Rate Limiting

OAuth callback endpoints are rate-limited:

```bash
# Configure rate limits
nself config set OAUTH_RATE_LIMIT_PER_MINUTE 60
nself config set OAUTH_RATE_LIMIT_PER_HOUR 1000
```

### Monitoring

Track OAuth performance:

```bash
# OAuth metrics
nself metrics oauth

# Output:
# oauth_logins_total{provider="google"} 1234
# oauth_logins_total{provider="github"} 567
# oauth_refresh_success_total 4567
# oauth_refresh_failure_total 12
```

---

## Compliance

### GDPR

OAuth data is personal data under GDPR:

```bash
# Export user's OAuth data
nself auth export-data <user_id>

# Delete user's OAuth accounts
nself auth delete-user <user_id> --include-oauth
```

### Data Retention

Configure OAuth data retention:

```bash
# Auto-delete unused OAuth accounts after 180 days
nself config set OAUTH_INACTIVE_ACCOUNT_RETENTION_DAYS 180

# Run cleanup
nself oauth cleanup --older-than 180d
```

---

## Related Commands

- [`nself auth`](./AUTH.md) - User authentication
- [`nself mfa`](./MFA.md) - Multi-factor authentication
- [`nself security`](./SECURITY.md) - Security settings
- [`nself audit`](./AUDIT.md) - Authentication audit logs

---

## Related Documentation

- [OAuth Complete Flows Guide](../guides/OAUTH-COMPLETE-FLOWS.md)
- [OAuth Setup Guide](../guides/OAUTH-SETUP.md)
- [OAuth API Reference](../api/OAUTH-API.md)
- [Provider Documentation](../providers/oauth/)
- [Security Best Practices](../security/README.md)

---

**Last Updated:** January 30, 2026
**Version:** 0.9.5
**Status:** Production Ready ✅
