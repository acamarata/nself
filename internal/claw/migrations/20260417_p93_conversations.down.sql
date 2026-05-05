-- Down migration for 20260417_p93_conversations.sql — np_claw_conversations table
-- Removes indexes, constraint, and table

DROP INDEX IF EXISTS idx_np_claw_conversations_last_message;
DROP INDEX IF EXISTS idx_np_claw_conversations_path;
DROP INDEX IF EXISTS idx_np_claw_conversations_topic_id;
DROP INDEX IF EXISTS idx_np_claw_conversations_user_id;
DROP TABLE IF EXISTS np_claw_conversations;
