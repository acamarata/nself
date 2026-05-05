-- P87 Sprint 37: Security Block G
-- Secrets rotation tracking, E2E encryption columns, SSO config, MFA, sessions

-- 1. Secrets rotation tracking table
CREATE TABLE IF NOT EXISTS np_secret_rotation (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    class           TEXT NOT NULL CHECK (class IN ('jwt','oauth','plugin_jwt','webhook','db','minio','api_key','custom')),
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','rotating','dual_key','retired','failed')),
    rotation_days   INT NOT NULL DEFAULT 90,
    grace_days      INT NOT NULL DEFAULT 7,
    last_rotated_at TIMESTAMPTZ,
    next_rotation_at TIMESTAMPTZ,
    dual_key_expires TIMESTAMPTZ,
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_np_secret_rotation_next ON np_secret_rotation(next_rotation_at) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_np_secret_rotation_dual ON np_secret_rotation(dual_key_expires) WHERE status = 'dual_key';

-- 2. E2E encryption columns on memories and messages
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS ciphertext BYTEA;
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS dek_wrapped BYTEA;
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS nonce BYTEA;
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS scheme TEXT CHECK (scheme IN ('xchacha20poly1305-v1', NULL));
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS blind_index BYTEA;

ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS ciphertext BYTEA;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS dek_wrapped BYTEA;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS nonce BYTEA;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS scheme TEXT CHECK (scheme IN ('xchacha20poly1305-v1', NULL));
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS blind_index BYTEA;

-- E2E user keys table
CREATE TABLE IF NOT EXISTS np_claw_user_keys (
    user_id         UUID PRIMARY KEY,
    kdf_params      JSONB NOT NULL DEFAULT '{}',
    public_key      BYTEA,
    e2e_enabled     BOOLEAN NOT NULL DEFAULT false,
    enabled_at      TIMESTAMPTZ,
    recovery_shards JSONB
);
CREATE INDEX IF NOT EXISTS idx_np_claw_user_keys_blind ON np_claw_memories(user_id, blind_index) WHERE blind_index IS NOT NULL;

-- 3. SSO configuration tables
CREATE TABLE IF NOT EXISTS np_sso_config (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID,
    provider_type   TEXT NOT NULL CHECK (provider_type IN ('google','microsoft','okta','oidc','saml')),
    client_id       TEXT NOT NULL,
    client_secret   TEXT,
    issuer_url      TEXT,
    metadata_url    TEXT,
    cert_pem        TEXT,
    entity_id       TEXT,
    scopes          TEXT[] DEFAULT ARRAY['openid','email','profile'],
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_sso_role_map (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id       UUID NOT NULL REFERENCES np_sso_config(id) ON DELETE CASCADE,
    idp_group       TEXT NOT NULL,
    nself_role      TEXT NOT NULL DEFAULT 'user',
    UNIQUE (config_id, idp_group)
);

-- 4. MFA tables
CREATE TABLE IF NOT EXISTS np_auth_mfa (
    user_id         UUID PRIMARY KEY,
    totp_secret     TEXT,
    totp_enabled    BOOLEAN NOT NULL DEFAULT false,
    totp_enabled_at TIMESTAMPTZ,
    webauthn_enabled BOOLEAN NOT NULL DEFAULT false,
    backup_codes    TEXT[],
    backup_count    INT NOT NULL DEFAULT 0,
    last_verified_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Session management tables
CREATE TABLE IF NOT EXISTS np_auth_devices (
    user_id    UUID NOT NULL,
    did        TEXT NOT NULL,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ua         TEXT NOT NULL DEFAULT '',
    ip         INET,
    label      TEXT,
    trusted    BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (user_id, did)
);

CREATE TABLE IF NOT EXISTS np_auth_refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    session_id UUID NOT NULL,
    did        TEXT NOT NULL DEFAULT '',
    chain_id   TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_np_auth_refresh_session ON np_auth_refresh_tokens(session_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_np_auth_refresh_chain ON np_auth_refresh_tokens(chain_id);
CREATE INDEX IF NOT EXISTS idx_np_auth_refresh_user ON np_auth_refresh_tokens(user_id) WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS np_auth_failed_logins (
    ip         INET NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_agent TEXT
);
CREATE INDEX IF NOT EXISTS idx_np_auth_failed_logins_ip ON np_auth_failed_logins(ip, attempted_at);
