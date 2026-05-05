-- DOWN
-- Rollback for 20260420_saved_searches.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_saved_searches_user;
DROP TABLE IF EXISTS np_claw_saved_searches CASCADE;
