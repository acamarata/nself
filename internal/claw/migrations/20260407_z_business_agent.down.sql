-- DOWN
-- Rollback for 20260407_z_business_agent.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_prompt_templates_persona;
ALTER TABLE IF EXISTS np_claw_prompt_templates DROP COLUMN IF EXISTS persona_name;
