-- DOWN
-- Rollback for 20260408_ai_model_scaffolds.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_image_jobs_user;
DROP TABLE IF EXISTS np_claw_image_jobs CASCADE;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS feature_flags;
DROP INDEX IF EXISTS idx_np_claw_routing_rules_active;
DROP INDEX IF EXISTS idx_np_claw_routing_rules_user;
DROP TABLE IF EXISTS np_claw_routing_rules CASCADE;
