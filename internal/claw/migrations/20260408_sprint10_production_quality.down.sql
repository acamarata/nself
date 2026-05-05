-- DOWN
-- Rollback for 20260408_sprint10_production_quality.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_agent_spend DROP COLUMN IF EXISTS category;
ALTER TABLE IF EXISTS np_claw.users DROP COLUMN IF EXISTS disabled;
DROP TABLE IF EXISTS np_claw_webhook_dlq CASCADE;
