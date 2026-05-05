-- Sprint 11: AI & Model Scaffolds
-- Adds routing_rules table, feature_flags column, image generation jobs

-- Routing rules for user-defined model routing
CREATE TABLE IF NOT EXISTS np_claw_routing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL DEFAULT 'default',
    name TEXT NOT NULL DEFAULT '',
    condition_type TEXT NOT NULL CHECK (condition_type IN ('keyword', 'context_length', 'persona', 'time_of_day')),
    condition_value TEXT NOT NULL DEFAULT '',
    target_model TEXT NOT NULL DEFAULT '',
    priority INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_routing_rules_user ON np_claw_routing_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_routing_rules_active ON np_claw_routing_rules(user_id, is_active, priority);

-- Feature flags per conversation (tools_enabled, web_search, code_execution, etc.)
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS feature_flags JSONB DEFAULT '{}';

-- Image generation jobs (async pattern)
CREATE TABLE IF NOT EXISTS np_claw_image_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL DEFAULT 'default',
    prompt TEXT NOT NULL,
    style TEXT,
    size TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    result_url TEXT,
    revised_prompt TEXT,
    provider TEXT,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_np_claw_image_jobs_user ON np_claw_image_jobs(user_id, created_at DESC);

-- Summary field already exists on np_claw_sessions from session package, no migration needed
