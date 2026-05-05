-- nself-claw: OpenAI-compatible API gateway schema (T-1158)
-- Idempotent — IF NOT EXISTS throughout

-- API Keys for the OpenAI-compatible gateway
CREATE TABLE IF NOT EXISTS np_claw_api_keys (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        TEXT NOT NULL,
    name           TEXT NOT NULL,
    key_hash       TEXT NOT NULL UNIQUE,          -- SHA-256 of full key
    key_prefix     TEXT NOT NULL,                 -- first 12 chars: "sk-nself-xxxx"
    is_active      BOOLEAN NOT NULL DEFAULT true,
    admin_allowed  BOOLEAN NOT NULL DEFAULT false,
    rpm_limit      INTEGER NOT NULL DEFAULT 300,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at   TIMESTAMPTZ,
    revoked_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_np_claw_api_keys_user_id ON np_claw_api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_api_keys_hash ON np_claw_api_keys(key_hash);

-- Per-key usage tracking
CREATE TABLE IF NOT EXISTS np_claw_api_usage (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id          UUID NOT NULL REFERENCES np_claw_api_keys(id) ON DELETE CASCADE,
    model           TEXT NOT NULL,
    prompt_tokens   INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd        DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    tier_source     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_np_claw_api_usage_key_id ON np_claw_api_usage(key_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_api_usage_created_at ON np_claw_api_usage(created_at DESC);

-- System prompts library
CREATE TABLE IF NOT EXISTS np_claw_system_prompts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    name        TEXT NOT NULL,
    content     TEXT NOT NULL,
    is_default  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_np_claw_system_prompts_user_id ON np_claw_system_prompts(user_id);
