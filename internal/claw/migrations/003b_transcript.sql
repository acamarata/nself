-- T-0291: Full conversation transcript persistence
-- Note: np_claw_messages table is created by the in-code migration in sessions.rs.
-- This file extends it with additional columns and indexes for transcript features.

ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS user_id TEXT;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS tool_calls JSONB;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS tool_call_id TEXT;

-- Extended role support: tool_call and tool_result in addition to existing user/assistant/system/tool
ALTER TABLE np_claw_messages DROP CONSTRAINT IF EXISTS np_claw_messages_role_check;
ALTER TABLE np_claw_messages ADD CONSTRAINT np_claw_messages_role_check
  CHECK (role IN ('user', 'assistant', 'tool_call', 'tool_result', 'system', 'tool'));

-- Index for per-user message queries (transcript retrieval)
CREATE INDEX IF NOT EXISTS idx_np_claw_messages_user ON np_claw_messages(user_id, created_at DESC)
  WHERE user_id IS NOT NULL;

-- GIN index for full-text search on message content
CREATE INDEX IF NOT EXISTS idx_np_claw_messages_content_fts
  ON np_claw_messages USING gin(to_tsvector('english', content))
  WHERE content IS NOT NULL AND content != '';
