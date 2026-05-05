-- P88 Sprint 19: Seed default budget caps for existing users.
-- Idempotent: uses ON CONFLICT DO NOTHING.
-- Up migration.

-- Seed global default: $2.00/day, $30.00/month
INSERT INTO np_claw_budget_caps (user_id, scope, tool_name, daily_usd, monthly_usd, warn_pct, hard_block)
SELECT DISTINCT user_id, 'global', NULL, 2.00, 30.00, 80, true
FROM np_claw_sessions
WHERE user_id IS NOT NULL
ON CONFLICT (user_id, scope, COALESCE(tool_name, '')) DO NOTHING;

-- Seed per-tool defaults: ai.* tools at $1.00/day
INSERT INTO np_claw_budget_caps (user_id, scope, tool_name, daily_usd, monthly_usd, warn_pct, hard_block)
SELECT DISTINCT s.user_id, 'per_tool', 'ai', 1.00, 20.00, 80, true
FROM np_claw_sessions s
WHERE s.user_id IS NOT NULL
ON CONFLICT (user_id, scope, COALESCE(tool_name, '')) DO NOTHING;

-- Seed per-tool defaults: browser.* tools at $0.20/day
INSERT INTO np_claw_budget_caps (user_id, scope, tool_name, daily_usd, monthly_usd, warn_pct, hard_block)
SELECT DISTINCT s.user_id, 'per_tool', 'browser', 0.20, 5.00, 80, true
FROM np_claw_sessions s
WHERE s.user_id IS NOT NULL
ON CONFLICT (user_id, scope, COALESCE(tool_name, '')) DO NOTHING;

-- Seed per-tool defaults: research.* tools at $0.50/day
INSERT INTO np_claw_budget_caps (user_id, scope, tool_name, daily_usd, monthly_usd, warn_pct, hard_block)
SELECT DISTINCT s.user_id, 'per_tool', 'research', 0.50, 10.00, 80, true
FROM np_claw_sessions s
WHERE s.user_id IS NOT NULL
ON CONFLICT (user_id, scope, COALESCE(tool_name, '')) DO NOTHING;

-- Down migration (reversibility):
-- DELETE FROM np_claw_budget_caps WHERE daily_usd IN (2.00, 1.00, 0.20, 0.50)
--   AND monthly_usd IN (30.00, 20.00, 5.00, 10.00);
