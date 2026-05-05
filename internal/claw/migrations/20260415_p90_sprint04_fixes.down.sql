-- DOWN
-- Rollback for 20260415_p90_sprint04_fixes.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP TABLE IF EXISTS np_claw_config CASCADE;
-- update np_claw_audit_log set actor_id = actor_user_id where ... cannot be auto-reversed; verify manually if rollback needed.
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS meta;
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS duration_ms;
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS error;
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS success;
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS after_state;
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS before_state;
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS actor_type;
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS actor_name;
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS actor_id;
-- update np_claw_audit_log set resource_id = target_id where r... cannot be auto-reversed; verify manually if rollback needed.
-- update np_claw_audit_log set resource_type = target_type whe... cannot be auto-reversed; verify manually if rollback needed.
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS resource_id;
ALTER TABLE IF EXISTS np_claw_audit_log DROP COLUMN IF EXISTS resource_type;
DROP TABLE IF EXISTS np_ai_model_catalog CASCADE;
