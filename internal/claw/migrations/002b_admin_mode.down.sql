-- DOWN
-- Rollback for 002b_admin_mode.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS is_admin_mode;
