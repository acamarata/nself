-- DOWN
-- Rollback for m_p88_c5_seed_builtins.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_retrieval_profiles DROP CONSTRAINT IF EXISTS np_claw_retrieval_profiles_query_class_check;
-- update np_claw_retrieval_profiles
set is_builtin = true, des... cannot be auto-reversed; verify manually if rollback needed.
