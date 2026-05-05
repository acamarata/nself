-- DOWN
-- Rollback for 013_sessions_drift_column.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS last_drift_suggested_at_count;
