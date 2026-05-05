-- DOWN
-- Rollback for 20260414_p87_audit_log_v2.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP TRIGGER IF EXISTS trg_audit_log_immutable ON np_claw_audit_log;
DROP FUNCTION IF EXISTS np_claw_audit_log_immutable() CASCADE;
DROP INDEX IF EXISTS idx_audit_log_meta;
DROP INDEX IF EXISTS idx_audit_log_resource;
-- update np_claw_audit_log set actor_id = actor_user_id where ... cannot be auto-reversed; verify manually if rollback needed.
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS cost_usd;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS duration_ms;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS error;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS success;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS meta;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS after_state;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS before_state;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS resource_id;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS resource_type;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS request_id;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS session_id;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS impersonator;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS actor_id;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS actor_type;
