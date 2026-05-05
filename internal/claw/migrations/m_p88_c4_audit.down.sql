-- DOWN
-- Rollback for m_p88_c4_audit.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_audit_export_rate_user;
DROP TABLE IF EXISTS np_claw_audit_export_rate CASCADE;
DROP TABLE IF EXISTS np_claw_audit_retention_config CASCADE;
DROP INDEX IF EXISTS idx_audit_action_created;
DROP INDEX IF EXISTS idx_audit_actor_created;
DROP INDEX IF EXISTS idx_audit_created_brin;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS actor_name;
