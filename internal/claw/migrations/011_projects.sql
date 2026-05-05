-- T-1099: np_claw_projects table for grouping sessions into named projects.
-- Idempotent — CREATE TABLE IF NOT EXISTS + ADD COLUMN IF NOT EXISTS throughout.
-- Also applied by the in-code migration in projects.rs migrate().

CREATE TABLE IF NOT EXISTS np_claw_projects (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT        NOT NULL,
    description     TEXT,
    color           TEXT,                       -- hex color e.g. "#6366F1"
    emoji           TEXT,                       -- single emoji e.g. "📁"
    system_prompt   TEXT,                       -- per-project system prompt override
    archived_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_projects_archived
    ON np_claw_projects(archived_at);

-- Partial index for active (non-archived) projects sorted by name
CREATE INDEX IF NOT EXISTS idx_np_claw_projects_active_name
    ON np_claw_projects(name ASC)
    WHERE archived_at IS NULL;

-- Add FK from sessions to projects (safe to run after both tables exist)
ALTER TABLE np_claw_sessions
    ADD CONSTRAINT IF NOT EXISTS fk_np_claw_sessions_project
    FOREIGN KEY (project_id) REFERENCES np_claw_projects(id)
    ON DELETE SET NULL
    NOT VALID;
