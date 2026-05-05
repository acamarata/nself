-- P88 Sprint 19: RLS policies for approval rules and budget tables.
-- Up migration.

-- RLS for approval rules
ALTER TABLE np_claw_approval_rules ENABLE ROW LEVEL SECURITY;

CREATE POLICY np_claw_approval_rules_user_policy ON np_claw_approval_rules
    USING (user_id = current_setting('hasura.user', true))
    WITH CHECK (user_id = current_setting('hasura.user', true));

-- RLS for budget caps
ALTER TABLE np_claw_budget_caps ENABLE ROW LEVEL SECURITY;

CREATE POLICY np_claw_budget_caps_user_policy ON np_claw_budget_caps
    USING (user_id = current_setting('hasura.user', true))
    WITH CHECK (user_id = current_setting('hasura.user', true));

-- RLS for budget usage
ALTER TABLE np_claw_budget_usage ENABLE ROW LEVEL SECURITY;

CREATE POLICY np_claw_budget_usage_user_policy ON np_claw_budget_usage
    USING (user_id = current_setting('hasura.user', true));

-- RLS for budget exceptions
ALTER TABLE np_claw_budget_exceptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY np_claw_budget_exceptions_user_policy ON np_claw_budget_exceptions
    USING (user_id = current_setting('hasura.user', true))
    WITH CHECK (user_id = current_setting('hasura.user', true));

-- Down migration (reversibility):
-- DROP POLICY IF EXISTS np_claw_budget_exceptions_user_policy ON np_claw_budget_exceptions;
-- ALTER TABLE np_claw_budget_exceptions DISABLE ROW LEVEL SECURITY;
-- DROP POLICY IF EXISTS np_claw_budget_usage_user_policy ON np_claw_budget_usage;
-- ALTER TABLE np_claw_budget_usage DISABLE ROW LEVEL SECURITY;
-- DROP POLICY IF EXISTS np_claw_budget_caps_user_policy ON np_claw_budget_caps;
-- ALTER TABLE np_claw_budget_caps DISABLE ROW LEVEL SECURITY;
-- DROP POLICY IF EXISTS np_claw_approval_rules_user_policy ON np_claw_approval_rules;
-- ALTER TABLE np_claw_approval_rules DISABLE ROW LEVEL SECURITY;
