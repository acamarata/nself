-- Research mode: stores research query results with sources, claims, citations
-- Note: table is also created programmatically by research.Migrate() on startup.
-- This file exists for migration tooling consistency.
CREATE TABLE IF NOT EXISTS np_claw_research_results (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    query        TEXT        NOT NULL,
    depth        TEXT        NOT NULL DEFAULT 'standard',
    synthesis    TEXT        NOT NULL DEFAULT '',
    sources      JSONB       NOT NULL DEFAULT '[]'::jsonb,
    claims       JSONB       NOT NULL DEFAULT '[]'::jsonb,
    citations    JSONB       NOT NULL DEFAULT '[]'::jsonb,
    review_notes JSONB       NOT NULL DEFAULT '[]'::jsonb,
    confidence   FLOAT       NOT NULL DEFAULT 0,
    model_used   TEXT        NOT NULL DEFAULT '',
    duration_ms  BIGINT      NOT NULL DEFAULT 0,
    user_id      TEXT,
    session_id   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_claw_research_user_id ON np_claw_research_results (user_id);
CREATE INDEX IF NOT EXISTS idx_claw_research_created ON np_claw_research_results (created_at DESC);
