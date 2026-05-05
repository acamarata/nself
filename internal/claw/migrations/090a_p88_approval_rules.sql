-- P88 Sprint 19: Auto-approve rules table + budget caps + budget exceptions.
-- Up migration.

-- Approval rules: predicate-based auto-approve
CREATE TABLE IF NOT EXISTS np_claw_approval_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             TEXT NOT NULL,
    tool                TEXT NOT NULL,
    match_predicate_json JSONB NOT NULL DEFAULT '{"all":[]}',
    max_cost_usd        NUMERIC(10,4) NOT NULL DEFAULT 0,
    max_per_hour        INT NOT NULL DEFAULT 0,
    enabled             BOOLEAN NOT NULL DEFAULT true,
    expires_at          TIMESTAMPTZ,
    last_hit_at         TIMESTAMPTZ,
    hit_count           BIGINT NOT NULL DEFAULT 0,
    requires_two_factor BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_claw_approval_rules_user_tool
    ON np_claw_approval_rules (user_id, tool) WHERE enabled = true;

-- Budget caps: per-user global and per-tool spending limits
CREATE TABLE IF NOT EXISTS np_claw_budget_caps (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    scope       TEXT NOT NULL CHECK (scope IN ('global', 'per_tool')),
    tool_name   TEXT,
    daily_usd   NUMERIC(10,4) NOT NULL DEFAULT 2.00,
    monthly_usd NUMERIC(10,4) NOT NULL DEFAULT 30.00,
    warn_pct    INT NOT NULL DEFAULT 80,
    hard_block  BOOLEAN NOT NULL DEFAULT true,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, scope, COALESCE(tool_name, ''))
);

-- Budget usage snapshot (Redis is primary, PG is nightly snapshot)
CREATE TABLE IF NOT EXISTS np_claw_budget_usage (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    scope       TEXT NOT NULL,
    tool_name   TEXT,
    date        DATE NOT NULL,
    usd_spent   NUMERIC(10,4) NOT NULL DEFAULT 0,
    calls_count INT NOT NULL DEFAULT 0,
    UNIQUE (user_id, scope, COALESCE(tool_name, ''), date)
);

-- Budget exception tokens: single-use override for cost cap
CREATE TABLE IF NOT EXISTS np_claw_budget_exceptions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    tool_name   TEXT NOT NULL,
    consumed    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_claw_budget_exceptions_active
    ON np_claw_budget_exceptions (user_id, tool_name) WHERE consumed = false;

-- Down migration (reversibility):
-- DROP TABLE IF EXISTS np_claw_budget_exceptions;
-- DROP TABLE IF EXISTS np_claw_budget_usage;
-- DROP TABLE IF EXISTS np_claw_budget_caps;
-- DROP TABLE IF EXISTS np_claw_approval_rules;
