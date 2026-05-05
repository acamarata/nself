-- DOWN
-- Rollback for 090a_p88_approval_rules.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_claw_budget_exceptions_active;
DROP TABLE IF EXISTS np_claw_budget_exceptions CASCADE;
DROP TABLE IF EXISTS np_claw_budget_usage CASCADE;
DROP TABLE IF EXISTS np_claw_budget_caps CASCADE;
DROP INDEX IF EXISTS idx_claw_approval_rules_user_tool;
DROP TABLE IF EXISTS np_claw_approval_rules CASCADE;
