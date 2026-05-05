-- Sprint 10: Production quality additions

-- Webhook dead letter queue for failed deliveries (T14)
CREATE TABLE IF NOT EXISTS np_claw_webhook_dlq (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id TEXT NOT NULL,
    event TEXT NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User disabled flag for admin management (T17)
ALTER TABLE np_claw.users ADD COLUMN IF NOT EXISTS disabled BOOLEAN DEFAULT FALSE;

-- Budget category column for spend tracking (QA-M10)
DO $$ BEGIN
    ALTER TABLE np_claw_agent_spend ADD COLUMN IF NOT EXISTS category TEXT;
EXCEPTION WHEN duplicate_column THEN NULL;
END $$;
