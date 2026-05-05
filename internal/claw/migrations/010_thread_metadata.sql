-- T-1098: Thread metadata columns on np_claw_sessions and np_claw_messages.
-- All idempotent — IF NOT EXISTS / ADD COLUMN IF NOT EXISTS throughout.
-- These columns are also applied by the in-code migration in sessions.rs migrate(),
-- so this file is safe to re-run on an already-migrated database.

ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS title TEXT;
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS auto_title TEXT;
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]';
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS project_id UUID;
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS parent_session_id UUID REFERENCES np_claw_sessions(id);
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS topic_fingerprint TEXT;
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS summary TEXT;
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS is_admin_mode BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMPTZ;
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS last_drift_suggested_at_count BIGINT;

-- GIN index for fulltext search on title + auto_title (T-1100)
CREATE INDEX IF NOT EXISTS idx_np_claw_sessions_fts
    ON np_claw_sessions
    USING GIN(to_tsvector('english', COALESCE(auto_title, '') || ' ' || COALESCE(title, '')));

-- Index for project-filtered queries
CREATE INDEX IF NOT EXISTS idx_np_claw_sessions_project
    ON np_claw_sessions(project_id)
    WHERE project_id IS NOT NULL;

-- Index for archived/active listing sorted by activity
CREATE INDEX IF NOT EXISTS idx_np_claw_sessions_active_listing
    ON np_claw_sessions(archived_at, last_message_at DESC NULLS LAST, updated_at DESC)
    WHERE archived_at IS NULL;

-- T-1098: Message telemetry columns on np_claw_messages
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS tier_source TEXT;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS model_used TEXT;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS latency_ms INTEGER;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS input_tokens INTEGER;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS output_tokens INTEGER;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';
