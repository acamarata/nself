-- P93 S02-T04: Add model_hint column to np_claw_personas.
ALTER TABLE np_claw_personas ADD COLUMN IF NOT EXISTS model_hint TEXT;
-- Rollback: ALTER TABLE np_claw_personas DROP COLUMN IF EXISTS model_hint;
