-- DOWN
-- Rollback for 20260408_sprint15_persona_features.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_agent_spend DROP COLUMN IF EXISTS tokens_used;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_persona_marketplace DROP COLUMN IF EXISTS rating_count;
-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_personas DROP COLUMN IF EXISTS category;
DROP TABLE IF EXISTS np_claw_domain_expert_config CASCADE;
DROP TABLE IF EXISTS np_claw_marketplace_installs CASCADE;
DROP INDEX IF EXISTS idx_claw_marketplace_ratings_persona;
DROP TABLE IF EXISTS np_claw_marketplace_ratings CASCADE;
DROP TABLE IF EXISTS np_claw_cv_data CASCADE;
DROP INDEX IF EXISTS idx_claw_reflections_user;
DROP TABLE IF EXISTS np_claw_reflections CASCADE;
DROP INDEX IF EXISTS idx_claw_habit_logs_habit;
DROP TABLE IF EXISTS np_claw_habit_logs CASCADE;
DROP INDEX IF EXISTS idx_claw_habits_user;
DROP TABLE IF EXISTS np_claw_habits CASCADE;
DROP INDEX IF EXISTS idx_claw_tutor_cards_due;
DROP TABLE IF EXISTS np_claw_tutor_cards CASCADE;
DROP INDEX IF EXISTS idx_claw_health_checkins_user;
DROP TABLE IF EXISTS np_claw_health_checkins CASCADE;
