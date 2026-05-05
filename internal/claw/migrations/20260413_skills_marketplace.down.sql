-- DOWN
-- Rollback for 20260413_skills_marketplace.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_skills_public;
ALTER TABLE IF EXISTS np_claw_skills DROP COLUMN IF EXISTS published_at;
ALTER TABLE IF EXISTS np_claw_skills DROP COLUMN IF EXISTS is_public;
