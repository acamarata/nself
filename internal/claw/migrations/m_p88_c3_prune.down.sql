-- DOWN
-- Rollback for m_p88_c3_prune.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_prune_policy_user;
DROP TABLE IF EXISTS np_claw_prune_policy CASCADE;
DROP INDEX IF EXISTS idx_prune_candidates_age;
DROP INDEX IF EXISTS idx_prune_candidates_filter;
ALTER TABLE IF EXISTS np_claw_prune_candidates DROP COLUMN IF EXISTS reason_note;
ALTER TABLE IF EXISTS np_claw_prune_candidates DROP COLUMN IF EXISTS rejected_at;
ALTER TABLE IF EXISTS np_claw_prune_candidates DROP COLUMN IF EXISTS defer_until;
