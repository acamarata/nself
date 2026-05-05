-- H50-07: Free-tier savings daily view.
-- Shows per-user daily savings from free-tier providers vs cheapest paid alternative.

CREATE OR REPLACE VIEW np_ai_free_tier_savings_daily AS
SELECT
    u.user_id,
    u.day,
    SUM(CASE
        WHEN u.provider IN ('ollama')
          OR (u.provider = 'google' AND u.routed_reason IN ('free_pool', 'oauth'))
        THEN fallback.cost_equiv_usd
        ELSE 0
    END) AS saved_usd,
    SUM(CASE
        WHEN u.provider NOT IN ('ollama')
          AND NOT (u.provider = 'google' AND u.routed_reason IN ('free_pool', 'oauth'))
        THEN u.cost_usd
        ELSE 0
    END) AS spent_usd
FROM np_ai_user_daily_cost u
LEFT JOIN LATERAL (
    SELECT (u.input_tokens / 1e6 * p2.input_per_mtok_usd
          + u.output_tokens / 1e6 * p2.output_per_mtok_usd) AS cost_equiv_usd
    FROM np_ai_provider_pricing p2
    WHERE p2.tier = 'paid'
      AND p2.deprecated_at IS NULL
    ORDER BY p2.input_per_mtok_usd ASC
    LIMIT 1
) fallback ON true
GROUP BY u.user_id, u.day;

-- H50-18: Cost override audit table (if not exists).
CREATE TABLE IF NOT EXISTS np_ai_cost_override_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_id UUID NOT NULL,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cost_override_audit_created
    ON np_ai_cost_override_audit (created_at DESC);

-- H50-11: Recommendation actioned table.
CREATE TABLE IF NOT EXISTS np_ai_recommendation_actioned (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    purpose TEXT NOT NULL,
    old_model TEXT NOT NULL,
    new_model TEXT NOT NULL,
    accepted BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recommendation_actioned_user
    ON np_ai_recommendation_actioned (user_id, created_at DESC);

-- H50-08: Free-tier quota table (if not exists).
CREATE TABLE IF NOT EXISTS np_ai_free_tier_quota (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    provider TEXT NOT NULL,
    tier TEXT NOT NULL,
    day DATE NOT NULL DEFAULT CURRENT_DATE,
    used_calls BIGINT NOT NULL DEFAULT 0,
    cap_calls BIGINT NOT NULL DEFAULT 1500,
    UNIQUE (user_id, provider, tier, day)
);
