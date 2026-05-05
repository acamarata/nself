-- DOWN
-- Rollback for m_p88_c5_profiles.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_profiles_active;
DROP INDEX IF EXISTS idx_quality_daily_profile;
DROP TABLE IF EXISTS np_claw_profile_quality_daily CASCADE;
DROP INDEX IF EXISTS idx_profile_ab_active;
DROP TABLE IF EXISTS np_claw_profile_ab_tests CASCADE;
ALTER TABLE IF EXISTS np_claw_retrieval_profiles DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE IF EXISTS np_claw_retrieval_profiles DROP COLUMN IF EXISTS description;
ALTER TABLE IF EXISTS np_claw_retrieval_profiles DROP COLUMN IF EXISTS max_results;
ALTER TABLE IF EXISTS np_claw_retrieval_profiles DROP COLUMN IF EXISTS score_floor;
ALTER TABLE IF EXISTS np_claw_retrieval_profiles DROP COLUMN IF EXISTS is_builtin;
