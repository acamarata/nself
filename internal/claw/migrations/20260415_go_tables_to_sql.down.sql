-- DOWN
-- Rollback for 20260415_go_tables_to_sql.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS room_id;
DROP INDEX IF EXISTS idx_memory_rooms_user;
DROP TABLE IF EXISTS np_claw_memory_rooms CASCADE;
DROP TABLE IF EXISTS np_claw_knowledge_edges CASCADE;
DROP TABLE IF EXISTS np_claw_knowledge_nodes CASCADE;
DROP INDEX IF EXISTS idx_np_claw_webauthn_state_expires;
DROP TABLE IF EXISTS np_claw.webauthn_state CASCADE;
DROP TABLE IF EXISTS np_claw.push_subscriptions CASCADE;
DROP TABLE IF EXISTS np_claw.setup_tokens CASCADE;
DROP INDEX IF EXISTS idx_np_claw_sessions_user;
DROP TABLE IF EXISTS np_claw.sessions CASCADE;
DROP TABLE IF EXISTS np_claw.passkeys CASCADE;
DROP TABLE IF EXISTS np_claw.users CASCADE;
-- DROP SCHEMA IF EXISTS np_claw CASCADE;  -- uncomment only if schema is exclusively owned by this migration
