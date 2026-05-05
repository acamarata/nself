-- DOWN
-- Rollback for 20260408_sprint17_voice_files_workflows.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_sessions_identity;
DROP INDEX IF EXISTS idx_np_claw_workflow_templates_public;
DROP INDEX IF EXISTS idx_np_claw_workflow_templates_share;
ALTER TABLE IF EXISTS np_claw_workflow_templates DROP COLUMN IF EXISTS updated_at;
ALTER TABLE IF EXISTS np_claw_workflow_templates DROP COLUMN IF EXISTS enabled;
ALTER TABLE IF EXISTS np_claw_workflow_templates DROP COLUMN IF EXISTS variables;
ALTER TABLE IF EXISTS np_claw_workflow_templates DROP COLUMN IF EXISTS share_token;
ALTER TABLE IF EXISTS np_claw_workflow_templates DROP COLUMN IF EXISTS is_public;
DROP INDEX IF EXISTS idx_np_claw_project_files_parent;
DROP INDEX IF EXISTS idx_np_claw_project_files_project;
DROP TABLE IF EXISTS np_claw_project_files CASCADE;
ALTER TABLE IF EXISTS np_claw_identities DROP COLUMN IF EXISTS voice_provider;
ALTER TABLE IF EXISTS np_claw_identities DROP COLUMN IF EXISTS voice_id;
