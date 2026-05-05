-- DOWN
-- Rollback for 20260407_prompt_templates.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_prompt_templates_builtin;
DROP INDEX IF EXISTS idx_np_claw_prompt_templates_user_id;
DROP INDEX IF EXISTS idx_np_claw_prompt_templates_category;
DROP TABLE IF EXISTS np_claw_prompt_templates CASCADE;
