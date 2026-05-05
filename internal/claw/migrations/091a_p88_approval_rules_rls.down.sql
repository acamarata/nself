-- DOWN
-- Rollback for 091a_p88_approval_rules_rls.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP POLICY IF EXISTS np_claw_budget_exceptions_user_policy ON np_claw_budget_exceptions;
ALTER TABLE IF EXISTS np_claw_budget_exceptions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS np_claw_budget_usage_user_policy ON np_claw_budget_usage;
ALTER TABLE IF EXISTS np_claw_budget_usage DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS np_claw_budget_caps_user_policy ON np_claw_budget_caps;
ALTER TABLE IF EXISTS np_claw_budget_caps DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS np_claw_approval_rules_user_policy ON np_claw_approval_rules;
ALTER TABLE IF EXISTS np_claw_approval_rules DISABLE ROW LEVEL SECURITY;
