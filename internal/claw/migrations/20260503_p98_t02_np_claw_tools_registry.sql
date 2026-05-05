-- P98 S09-T02: Create np_claw_tools_registry — spec gap (Chain-eddf35a6).
-- Tracks which tools (browser, google, voice, etc.) are registered for a user's
-- claw session. Described in claw plugin spec but never implemented as a migration.

CREATE TABLE IF NOT EXISTS np_claw_tools_registry (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            TEXT NOT NULL,
    tool_name          TEXT NOT NULL,
    enabled            BOOLEAN NOT NULL DEFAULT true,
    config             JSONB NOT NULL DEFAULT '{}',
    source_account_id  TEXT NOT NULL DEFAULT 'primary',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_np_claw_tools_registry_user_tool_account
        UNIQUE (user_id, tool_name, source_account_id)
);

CREATE INDEX IF NOT EXISTS idx_np_claw_tools_registry_user
    ON np_claw_tools_registry(user_id, source_account_id);

CREATE INDEX IF NOT EXISTS idx_np_claw_tools_registry_enabled
    ON np_claw_tools_registry(tool_name, enabled)
    WHERE enabled = true;
