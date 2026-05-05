-- DOWN
-- Rollback for 010_thread_metadata.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS metadata;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS output_tokens;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS input_tokens;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS latency_ms;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS model_used;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS tier_source;
DROP INDEX IF EXISTS idx_np_claw_sessions_active_listing;
DROP INDEX IF EXISTS idx_np_claw_sessions_project;
DROP INDEX IF EXISTS idx_np_claw_sessions_fts;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS last_drift_suggested_at_count;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS last_message_at;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS archived_at;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS is_admin_mode;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS summary;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS topic_fingerprint;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS parent_session_id;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS project_id;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS tags;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS auto_title;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS title;
