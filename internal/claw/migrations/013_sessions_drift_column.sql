-- T-1512: Ensure last_drift_suggested_at_count column exists
-- This column may be missing if the inline ALTER TABLE in sessions.rs failed silently.
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS last_drift_suggested_at_count BIGINT;
