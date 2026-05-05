-- DOWN
-- Rollback for 20260417_p93_persona_model_hint.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_personas DROP COLUMN IF EXISTS model_hint;
