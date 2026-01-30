-- OAuth Database Migrations for nself
-- Complete OAuth provider integration with Hasura Auth
-- Version: 1.0.0
-- Date: 2026-01-30

-- ============================================================================
-- OAuth Provider Accounts Table
-- ============================================================================
-- Stores OAuth provider connections for each user
-- Supports multiple providers per user (e.g., Google + GitHub)

CREATE TABLE IF NOT EXISTS auth.oauth_provider_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  provider VARCHAR(50) NOT NULL,
  provider_user_id VARCHAR(255) NOT NULL,
  provider_account_email VARCHAR(255),
  access_token TEXT,
  refresh_token TEXT,
  token_expires_at TIMESTAMPTZ,
  id_token TEXT,
  scopes TEXT[],
  raw_profile JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Constraints
  UNIQUE(provider, provider_user_id),
  UNIQUE(user_id, provider)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_oauth_provider_accounts_user_id
  ON auth.oauth_provider_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_provider_accounts_provider
  ON auth.oauth_provider_accounts(provider);
CREATE INDEX IF NOT EXISTS idx_oauth_provider_accounts_expires_at
  ON auth.oauth_provider_accounts(token_expires_at)
  WHERE token_expires_at IS NOT NULL;

-- Updated timestamp trigger
CREATE OR REPLACE FUNCTION auth.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER oauth_provider_accounts_updated_at
  BEFORE UPDATE ON auth.oauth_provider_accounts
  FOR EACH ROW
  EXECUTE FUNCTION auth.set_updated_at();

-- ============================================================================
-- OAuth State Table
-- ============================================================================
-- Stores OAuth state parameters for CSRF protection
-- Short-lived entries (10 minutes TTL)

CREATE TABLE IF NOT EXISTS auth.oauth_states (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  state VARCHAR(64) UNIQUE NOT NULL,
  provider VARCHAR(50) NOT NULL,
  redirect_url TEXT,
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '10 minutes')
);

-- Index for state lookup
CREATE INDEX IF NOT EXISTS idx_oauth_states_state
  ON auth.oauth_states(state);
CREATE INDEX IF NOT EXISTS idx_oauth_states_expires_at
  ON auth.oauth_states(expires_at);

-- Cleanup expired states function
CREATE OR REPLACE FUNCTION auth.cleanup_expired_oauth_states()
RETURNS void AS $$
BEGIN
  DELETE FROM auth.oauth_states WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- OAuth Token Refresh Queue
-- ============================================================================
-- Tracks tokens that need refresh
-- Processed by background worker

CREATE TABLE IF NOT EXISTS auth.oauth_token_refresh_queue (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  oauth_account_id UUID NOT NULL REFERENCES auth.oauth_provider_accounts(id) ON DELETE CASCADE,
  scheduled_at TIMESTAMPTZ NOT NULL,
  last_attempt_at TIMESTAMPTZ,
  attempts INT NOT NULL DEFAULT 0,
  max_attempts INT NOT NULL DEFAULT 3,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_oauth_token_refresh_queue_scheduled_at
  ON auth.oauth_token_refresh_queue(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_oauth_token_refresh_queue_account_id
  ON auth.oauth_token_refresh_queue(oauth_account_id);

-- ============================================================================
-- OAuth Provider Metadata Table
-- ============================================================================
-- Static configuration for OAuth providers

CREATE TABLE IF NOT EXISTS auth.oauth_providers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(50) UNIQUE NOT NULL,
  display_name VARCHAR(100) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT false,
  authorization_url TEXT NOT NULL,
  token_url TEXT NOT NULL,
  userinfo_url TEXT,
  revoke_url TEXT,
  default_scopes TEXT[] NOT NULL DEFAULT '{}',
  icon_url TEXT,
  color VARCHAR(7),
  supports_refresh BOOLEAN NOT NULL DEFAULT true,
  requires_pkce BOOLEAN NOT NULL DEFAULT false,
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER oauth_providers_updated_at
  BEFORE UPDATE ON auth.oauth_providers
  FOR EACH ROW
  EXECUTE FUNCTION auth.set_updated_at();

-- Insert default providers
INSERT INTO auth.oauth_providers (name, display_name, authorization_url, token_url, userinfo_url, revoke_url, default_scopes, icon_url, color, supports_refresh)
VALUES
  (
    'google',
    'Google',
    'https://accounts.google.com/o/oauth2/v2/auth',
    'https://oauth2.googleapis.com/token',
    'https://www.googleapis.com/oauth2/v2/userinfo',
    'https://oauth2.googleapis.com/revoke',
    ARRAY['openid', 'profile', 'email'],
    'https://www.google.com/favicon.ico',
    '#4285F4',
    true
  ),
  (
    'github',
    'GitHub',
    'https://github.com/login/oauth/authorize',
    'https://github.com/login/oauth/access_token',
    'https://api.github.com/user',
    'https://github.com/settings/connections/applications',
    ARRAY['read:user', 'user:email'],
    'https://github.com/favicon.ico',
    '#24292e',
    false
  ),
  (
    'microsoft',
    'Microsoft',
    'https://login.microsoftonline.com/common/oauth2/v2.0/authorize',
    'https://login.microsoftonline.com/common/oauth2/v2.0/token',
    'https://graph.microsoft.com/v1.0/me',
    'https://login.microsoftonline.com/common/oauth2/v2.0/logout',
    ARRAY['openid', 'profile', 'email'],
    'https://www.microsoft.com/favicon.ico',
    '#00A4EF',
    true
  ),
  (
    'slack',
    'Slack',
    'https://slack.com/oauth/v2/authorize',
    'https://slack.com/api/oauth.v2.access',
    'https://slack.com/api/users.identity',
    null,
    ARRAY['openid', 'profile', 'email'],
    'https://slack.com/favicon.ico',
    '#611f69',
    true
  ),
  (
    'discord',
    'Discord',
    'https://discord.com/api/oauth2/authorize',
    'https://discord.com/api/oauth2/token',
    'https://discord.com/api/users/@me',
    'https://discord.com/api/oauth2/token/revoke',
    ARRAY['identify', 'email'],
    'https://discord.com/favicon.ico',
    '#5865F2',
    true
  ),
  (
    'gitlab',
    'GitLab',
    'https://gitlab.com/oauth/authorize',
    'https://gitlab.com/oauth/token',
    'https://gitlab.com/api/v4/user',
    'https://gitlab.com/oauth/revoke',
    ARRAY['read_user', 'email'],
    'https://gitlab.com/favicon.ico',
    '#FC6D26',
    true
  ),
  (
    'bitbucket',
    'Bitbucket',
    'https://bitbucket.org/site/oauth2/authorize',
    'https://bitbucket.org/site/oauth2/access_token',
    'https://api.bitbucket.org/2.0/user',
    null,
    ARRAY['account', 'email'],
    'https://bitbucket.org/favicon.ico',
    '#0052CC',
    true
  ),
  (
    'twitter',
    'Twitter/X',
    'https://twitter.com/i/oauth2/authorize',
    'https://api.twitter.com/2/oauth2/token',
    'https://api.twitter.com/2/users/me',
    'https://api.twitter.com/2/oauth2/revoke',
    ARRAY['tweet.read', 'users.read'],
    'https://twitter.com/favicon.ico',
    '#1DA1F2',
    true
  ),
  (
    'facebook',
    'Facebook',
    'https://www.facebook.com/v18.0/dialog/oauth',
    'https://graph.facebook.com/v18.0/oauth/access_token',
    'https://graph.facebook.com/me',
    'https://graph.facebook.com/me/permissions',
    ARRAY['public_profile', 'email'],
    'https://www.facebook.com/favicon.ico',
    '#1877F2',
    true
  ),
  (
    'apple',
    'Apple',
    'https://appleid.apple.com/auth/authorize',
    'https://appleid.apple.com/auth/token',
    null,
    'https://appleid.apple.com/auth/revoke',
    ARRAY['name', 'email'],
    'https://www.apple.com/favicon.ico',
    '#000000',
    true
  ),
  (
    'linkedin',
    'LinkedIn',
    'https://www.linkedin.com/oauth/v2/authorization',
    'https://www.linkedin.com/oauth/v2/accessToken',
    'https://api.linkedin.com/v2/me',
    null,
    ARRAY['r_liteprofile', 'r_emailaddress'],
    'https://www.linkedin.com/favicon.ico',
    '#0077B5',
    true
  ),
  (
    'twitch',
    'Twitch',
    'https://id.twitch.tv/oauth2/authorize',
    'https://id.twitch.tv/oauth2/token',
    'https://api.twitch.tv/helix/users',
    'https://id.twitch.tv/oauth2/revoke',
    ARRAY['user:read:email'],
    'https://www.twitch.tv/favicon.ico',
    '#9146FF',
    true
  ),
  (
    'spotify',
    'Spotify',
    'https://accounts.spotify.com/authorize',
    'https://accounts.spotify.com/api/token',
    'https://api.spotify.com/v1/me',
    null,
    ARRAY['user-read-email', 'user-read-private'],
    'https://www.spotify.com/favicon.ico',
    '#1DB954',
    true
  )
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- OAuth Audit Log
-- ============================================================================
-- Track OAuth authentication events

CREATE TABLE IF NOT EXISTS auth.oauth_audit_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
  provider VARCHAR(50) NOT NULL,
  event_type VARCHAR(50) NOT NULL, -- 'login', 'link', 'unlink', 'refresh', 'revoke'
  ip_address INET,
  user_agent TEXT,
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_oauth_audit_log_user_id
  ON auth.oauth_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_audit_log_provider
  ON auth.oauth_audit_log(provider);
CREATE INDEX IF NOT EXISTS idx_oauth_audit_log_created_at
  ON auth.oauth_audit_log(created_at);

-- ============================================================================
-- Helper Functions
-- ============================================================================

-- Find or create OAuth user
CREATE OR REPLACE FUNCTION auth.find_or_create_oauth_user(
  p_provider VARCHAR(50),
  p_provider_user_id VARCHAR(255),
  p_email VARCHAR(255),
  p_display_name VARCHAR(255),
  p_avatar_url TEXT,
  p_raw_profile JSONB
)
RETURNS UUID AS $$
DECLARE
  v_user_id UUID;
  v_existing_account UUID;
BEGIN
  -- Check if OAuth account already exists
  SELECT user_id INTO v_user_id
  FROM auth.oauth_provider_accounts
  WHERE provider = p_provider AND provider_user_id = p_provider_user_id;

  -- If account exists, return user_id
  IF v_user_id IS NOT NULL THEN
    RETURN v_user_id;
  END IF;

  -- Check if user exists by email
  SELECT id INTO v_user_id
  FROM auth.users
  WHERE email = p_email;

  -- If user doesn't exist, create new user
  IF v_user_id IS NULL THEN
    INSERT INTO auth.users (
      email,
      display_name,
      avatar_url,
      email_verified,
      disabled
    )
    VALUES (
      p_email,
      p_display_name,
      p_avatar_url,
      true, -- OAuth emails are pre-verified
      false
    )
    RETURNING id INTO v_user_id;
  END IF;

  RETURN v_user_id;
END;
$$ LANGUAGE plpgsql;

-- Store OAuth tokens
CREATE OR REPLACE FUNCTION auth.store_oauth_tokens(
  p_user_id UUID,
  p_provider VARCHAR(50),
  p_provider_user_id VARCHAR(255),
  p_provider_account_email VARCHAR(255),
  p_access_token TEXT,
  p_refresh_token TEXT,
  p_expires_in INT,
  p_id_token TEXT,
  p_scopes TEXT[],
  p_raw_profile JSONB
)
RETURNS UUID AS $$
DECLARE
  v_account_id UUID;
  v_expires_at TIMESTAMPTZ;
BEGIN
  -- Calculate expiration time
  IF p_expires_in IS NOT NULL THEN
    v_expires_at := NOW() + (p_expires_in || ' seconds')::INTERVAL;
  END IF;

  -- Insert or update OAuth account
  INSERT INTO auth.oauth_provider_accounts (
    user_id,
    provider,
    provider_user_id,
    provider_account_email,
    access_token,
    refresh_token,
    token_expires_at,
    id_token,
    scopes,
    raw_profile
  )
  VALUES (
    p_user_id,
    p_provider,
    p_provider_user_id,
    p_provider_account_email,
    p_access_token,
    p_refresh_token,
    v_expires_at,
    p_id_token,
    p_scopes,
    p_raw_profile
  )
  ON CONFLICT (user_id, provider) DO UPDATE SET
    access_token = EXCLUDED.access_token,
    refresh_token = EXCLUDED.refresh_token,
    token_expires_at = EXCLUDED.token_expires_at,
    id_token = EXCLUDED.id_token,
    scopes = EXCLUDED.scopes,
    raw_profile = EXCLUDED.raw_profile,
    updated_at = NOW()
  RETURNING id INTO v_account_id;

  -- Schedule token refresh if refresh_token is provided and token expires
  IF p_refresh_token IS NOT NULL AND v_expires_at IS NOT NULL THEN
    INSERT INTO auth.oauth_token_refresh_queue (
      oauth_account_id,
      scheduled_at
    )
    VALUES (
      v_account_id,
      v_expires_at - INTERVAL '5 minutes' -- Refresh 5 minutes before expiry
    )
    ON CONFLICT (oauth_account_id) DO UPDATE SET
      scheduled_at = EXCLUDED.scheduled_at,
      attempts = 0,
      error_message = NULL;
  END IF;

  RETURN v_account_id;
END;
$$ LANGUAGE plpgsql;

-- Unlink OAuth provider
CREATE OR REPLACE FUNCTION auth.unlink_oauth_provider(
  p_user_id UUID,
  p_provider VARCHAR(50)
)
RETURNS BOOLEAN AS $$
DECLARE
  v_deleted BOOLEAN;
BEGIN
  DELETE FROM auth.oauth_provider_accounts
  WHERE user_id = p_user_id AND provider = p_provider;

  GET DIAGNOSTICS v_deleted = ROW_COUNT;
  RETURN v_deleted > 0;
END;
$$ LANGUAGE plpgsql;

-- Get OAuth providers for user
CREATE OR REPLACE FUNCTION auth.get_user_oauth_providers(p_user_id UUID)
RETURNS TABLE (
  provider VARCHAR(50),
  provider_account_email VARCHAR(255),
  linked_at TIMESTAMPTZ,
  token_expires_at TIMESTAMPTZ
) AS $$
BEGIN
  RETURN QUERY
  SELECT
    opa.provider,
    opa.provider_account_email,
    opa.created_at,
    opa.token_expires_at
  FROM auth.oauth_provider_accounts opa
  WHERE opa.user_id = p_user_id
  ORDER BY opa.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Hasura Permissions
-- ============================================================================

-- Allow users to view their own OAuth accounts
GRANT SELECT ON auth.oauth_provider_accounts TO hasura_user;
GRANT SELECT ON auth.oauth_providers TO hasura_user;
GRANT SELECT ON auth.oauth_audit_log TO hasura_user;

-- Allow Hasura admin to manage OAuth accounts
GRANT ALL ON auth.oauth_provider_accounts TO hasura_admin;
GRANT ALL ON auth.oauth_states TO hasura_admin;
GRANT ALL ON auth.oauth_token_refresh_queue TO hasura_admin;
GRANT ALL ON auth.oauth_providers TO hasura_admin;
GRANT ALL ON auth.oauth_audit_log TO hasura_admin;

-- ============================================================================
-- Scheduled Job: Cleanup expired OAuth states
-- ============================================================================
-- Run every 5 minutes to remove expired state entries

-- Note: This would typically be set up with pg_cron or a background worker
-- Example with pg_cron (if enabled):
-- SELECT cron.schedule('cleanup-oauth-states', '*/5 * * * *', 'SELECT auth.cleanup_expired_oauth_states()');

COMMENT ON TABLE auth.oauth_provider_accounts IS 'OAuth provider connections for users';
COMMENT ON TABLE auth.oauth_states IS 'Temporary OAuth state storage for CSRF protection';
COMMENT ON TABLE auth.oauth_token_refresh_queue IS 'Queue for OAuth token refresh operations';
COMMENT ON TABLE auth.oauth_providers IS 'OAuth provider metadata and configuration';
COMMENT ON TABLE auth.oauth_audit_log IS 'OAuth authentication event audit log';
