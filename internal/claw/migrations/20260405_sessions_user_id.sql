-- nself-claw: add missing user_id column to np_claw_sessions (T-fix)
-- Idempotent — IF NOT EXISTS throughout

ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';

CREATE INDEX IF NOT EXISTS idx_np_claw_sessions_user_id
    ON np_claw_sessions(user_id);
