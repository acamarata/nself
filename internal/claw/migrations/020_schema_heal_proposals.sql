-- nself-claw: self-healing schema proposals (B37)
-- Tracks proposed additive migrations from AI-detected query errors.

CREATE TABLE IF NOT EXISTS np_schema_heal_proposals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    error_json      JSONB NOT NULL,            -- original Hasura validation-failed error
    query_sql       TEXT NOT NULL,             -- original query that triggered the error
    migration       TEXT NOT NULL,             -- proposed SQL (ADD COLUMN / CREATE TABLE only)
    status          TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','approved','rejected','applied','rolled_back')),
    applied_at      TIMESTAMPTZ,
    rolled_back_at  TIMESTAMPTZ,
    ai_confidence   NUMERIC(3,2)               -- 0.00–1.00; proposal is surfaced only if >= 0.80
);

CREATE INDEX IF NOT EXISTS idx_np_schema_heal_status
    ON np_schema_heal_proposals(status);

CREATE INDEX IF NOT EXISTS idx_np_schema_heal_created_at
    ON np_schema_heal_proposals(created_at DESC);

COMMENT ON TABLE np_schema_heal_proposals IS
    'Self-healing schema agent — additive migration proposals (B37). '
    'Only ADD COLUMN / CREATE TABLE SQL is ever stored here. '
    'DROP, TRUNCATE, ALTER TYPE are forbidden by the heal agent guard.';
