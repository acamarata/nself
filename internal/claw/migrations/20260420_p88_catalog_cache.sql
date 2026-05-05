-- P88 Block D: Tool catalog cache table
-- Stores pre-rendered catalog JSON per user, invalidated on creds/tier/mock changes.

CREATE TABLE IF NOT EXISTS np_claw_tool_catalog_cache (
    user_id         TEXT        PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    catalog_json    JSONB       NOT NULL DEFAULT '{}',
    catalog_version TEXT        NOT NULL DEFAULT '1.0.0'
);

-- RLS: users can only read their own cache
ALTER TABLE np_claw_tool_catalog_cache ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_tool_catalog_cache' AND policyname = 'catalog_cache_user_isolation'
    ) THEN
        CREATE POLICY catalog_cache_user_isolation ON np_claw_tool_catalog_cache
            USING (user_id = current_setting('request.jwt.claims', true)::json->>'sub');
    END IF;
END $$;

-- Add mock_mode column to user preferences if not exists
ALTER TABLE np_claw_user_preferences ADD COLUMN IF NOT EXISTS mock_mode BOOLEAN DEFAULT NULL;

-- Down migration (reverse):
-- DROP POLICY IF EXISTS catalog_cache_user_isolation ON np_claw_tool_catalog_cache;
-- DROP TABLE IF EXISTS np_claw_tool_catalog_cache;
-- ALTER TABLE np_claw_user_preferences DROP COLUMN IF EXISTS mock_mode;
