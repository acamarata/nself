-- P90 Sprint 04: Complete Features — fix missing tables, columns, seed data.

-- ============================================================================
-- 7. np_ai_model_catalog — CREATE + seed 10 models
-- ============================================================================
CREATE TABLE IF NOT EXISTS np_ai_model_catalog (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    provider        TEXT        NOT NULL,
    model           TEXT        NOT NULL,
    display_name    TEXT        NOT NULL,
    modality        TEXT        NOT NULL DEFAULT 'text',
    context_window  INTEGER     NOT NULL DEFAULT 0,
    max_output      INTEGER     NOT NULL DEFAULT 0,
    tier            TEXT        NOT NULL DEFAULT 'free',
    quality_score   NUMERIC(3,2) NOT NULL DEFAULT 0.70,
    cost_input_mtok NUMERIC(10,4) NOT NULL DEFAULT 0,
    cost_output_mtok NUMERIC(10,4) NOT NULL DEFAULT 0,
    supports_vision BOOLEAN     NOT NULL DEFAULT false,
    supports_tools  BOOLEAN     NOT NULL DEFAULT false,
    supports_streaming BOOLEAN  NOT NULL DEFAULT true,
    is_available    BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, model)
);

INSERT INTO np_ai_model_catalog (provider, model, display_name, modality, context_window, max_output, tier, quality_score, cost_input_mtok, cost_output_mtok, supports_vision, supports_tools)
VALUES
    ('anthropic', 'claude-sonnet-4-20250514', 'Claude Sonnet 4', 'text', 200000, 64000, 'pro', 0.92, 3.00, 15.00, true, true),
    ('anthropic', 'claude-haiku-4-20250514', 'Claude Haiku 4', 'text', 200000, 64000, 'basic', 0.80, 0.80, 4.00, true, true),
    ('anthropic', 'claude-opus-4-20250514', 'Claude Opus 4', 'text', 200000, 32000, 'pro', 0.98, 15.00, 75.00, true, true),
    ('openai', 'gpt-4o', 'GPT-4o', 'text', 128000, 16384, 'pro', 0.90, 2.50, 10.00, true, true),
    ('openai', 'gpt-4o-mini', 'GPT-4o Mini', 'text', 128000, 16384, 'basic', 0.78, 0.15, 0.60, true, true),
    ('google', 'gemini-2.5-flash', 'Gemini 2.5 Flash', 'text', 1000000, 65536, 'basic', 0.82, 0.15, 0.60, true, true),
    ('google', 'gemini-2.5-pro', 'Gemini 2.5 Pro', 'text', 1000000, 65536, 'pro', 0.91, 1.25, 10.00, true, true),
    ('ollama', 'llama3.1:8b', 'Llama 3.1 8B', 'text', 128000, 8192, 'free', 0.65, 0, 0, false, false),
    ('ollama', 'qwen2.5:14b', 'Qwen 2.5 14B', 'text', 128000, 8192, 'free', 0.70, 0, 0, false, true),
    ('ollama', 'mistral:7b', 'Mistral 7B', 'text', 32000, 4096, 'free', 0.62, 0, 0, false, false)
ON CONFLICT (provider, model) DO NOTHING;

-- ============================================================================
-- 11. Seed np_ai_provider_pricing so cost_usd is computed
-- ============================================================================
INSERT INTO np_ai_provider_pricing (provider, model, display_name, input_per_mtok_usd, output_per_mtok_usd, cache_read_per_mtok, cache_write_per_mtok, context_window, max_output_tokens, tier, quality_score, effective_date)
VALUES
    ('anthropic', 'claude-sonnet-4-20250514', 'Claude Sonnet 4', 3.00, 15.00, 0.30, 3.75, 200000, 64000, 'pro', 0.92, CURRENT_DATE),
    ('anthropic', 'claude-haiku-4-20250514', 'Claude Haiku 4', 0.80, 4.00, 0.08, 1.00, 200000, 64000, 'basic', 0.80, CURRENT_DATE),
    ('anthropic', 'claude-opus-4-20250514', 'Claude Opus 4', 15.00, 75.00, 1.50, 18.75, 200000, 32000, 'pro', 0.98, CURRENT_DATE),
    ('openai', 'gpt-4o', 'GPT-4o', 2.50, 10.00, 1.25, 2.50, 128000, 16384, 'pro', 0.90, CURRENT_DATE),
    ('openai', 'gpt-4o-mini', 'GPT-4o Mini', 0.15, 0.60, 0.075, 0.15, 128000, 16384, 'basic', 0.78, CURRENT_DATE),
    ('google', 'gemini-2.5-flash', 'Gemini 2.5 Flash', 0.15, 0.60, 0.0375, 0.15, 1000000, 65536, 'basic', 0.82, CURRENT_DATE),
    ('google', 'gemini-2.5-pro', 'Gemini 2.5 Pro', 1.25, 10.00, 0.3125, 1.25, 1000000, 65536, 'pro', 0.91, CURRENT_DATE),
    ('ollama', 'llama3.1:8b', 'Llama 3.1 8B', 0, 0, 0, 0, 128000, 8192, 'free', 0.65, CURRENT_DATE),
    ('ollama', 'qwen2.5:14b', 'Qwen 2.5 14B', 0, 0, 0, 0, 128000, 8192, 'free', 0.70, CURRENT_DATE),
    ('ollama', 'mistral:7b', 'Mistral 7B', 0, 0, 0, 0, 32000, 4096, 'free', 0.62, CURRENT_DATE)
ON CONFLICT (provider, model, effective_date) DO NOTHING;

-- ============================================================================
-- 12. Fix audit_log: add resource_type/resource_id aliases
-- ============================================================================
-- The schema has target_type/target_id but code (audit/export.go, writer.go)
-- references resource_type/resource_id. Add the missing columns.
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS resource_type TEXT;
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS resource_id TEXT;

-- Backfill resource_type from target_type for existing rows
UPDATE np_claw_audit_log SET resource_type = target_type WHERE resource_type IS NULL AND target_type IS NOT NULL;
UPDATE np_claw_audit_log SET resource_id = target_id WHERE resource_id IS NULL AND target_id IS NOT NULL;

-- Also add missing columns that the writer.go expects
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS actor_id TEXT;
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS actor_name TEXT;
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS actor_type TEXT NOT NULL DEFAULT 'user';
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS before_state JSONB;
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS after_state JSONB;
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS success BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS error TEXT;
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS duration_ms INTEGER;
ALTER TABLE np_claw_audit_log ADD COLUMN IF NOT EXISTS meta JSONB NOT NULL DEFAULT '{}';

-- Backfill actor_id from actor_user_id
UPDATE np_claw_audit_log SET actor_id = actor_user_id WHERE actor_id IS NULL AND actor_user_id IS NOT NULL;

-- ============================================================================
-- 13. Ensure np_claw_config table exists (may have been created by nSelf base)
-- ============================================================================
CREATE TABLE IF NOT EXISTS np_claw_config (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
