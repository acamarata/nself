-- Sprint 16: AI usage cost tracking for token savings display (T16.9)
-- Tracks per-request costs for savings calculation.

CREATE TABLE IF NOT EXISTS np_claw_ai_usage (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID REFERENCES np_claw_sessions(id) ON DELETE SET NULL,
    user_id         UUID,
    model_used      TEXT NOT NULL DEFAULT '',
    tier_used       TEXT NOT NULL DEFAULT '',
    input_tokens    INTEGER NOT NULL DEFAULT 0,
    output_tokens   INTEGER NOT NULL DEFAULT 0,
    estimated_full_cost NUMERIC(12,8) NOT NULL DEFAULT 0,
    actual_cost     NUMERIC(12,8) NOT NULL DEFAULT 0,
    routing_rule_id UUID,
    routing_rule_name TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_ai_usage_created ON np_claw_ai_usage (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_np_claw_ai_usage_session ON np_claw_ai_usage (session_id);

-- Add metadata JSONB column to sessions if not present (for summarize-and-fork linking)
DO $$ BEGIN
    ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';
EXCEPTION WHEN duplicate_column THEN NULL;
END $$;
