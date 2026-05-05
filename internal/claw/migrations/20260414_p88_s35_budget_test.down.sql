-- DOWN
-- Rollback for 20260414_p88_s35_budget_test.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_ai_task_class_stats DROP COLUMN IF EXISTS percentile_75_cost;
ALTER TABLE IF EXISTS np_ai_task_class_stats DROP COLUMN IF EXISTS percentile_50_cost;
ALTER TABLE IF EXISTS np_ai_task_class_stats DROP COLUMN IF EXISTS percentile_25_cost;
DROP INDEX IF EXISTS idx_notify_jobs_pending;
DROP TABLE IF EXISTS np_notify_jobs CASCADE;
DROP TABLE IF EXISTS np_ai_cost_override_audit CASCADE;
-- (derived from DO block — review manually)
DROP POLICY IF EXISTS notify_pref_user_isolation ON np_ai_notify_pref;
ALTER TABLE IF EXISTS np_ai_notify_pref DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS np_ai_notify_pref CASCADE;
DROP INDEX IF EXISTS idx_ai_budget_test_expiry;
ALTER TABLE IF EXISTS np_ai_budget DROP COLUMN IF EXISTS test_expires_at;
ALTER TABLE IF EXISTS np_ai_budget DROP COLUMN IF EXISTS is_test;
