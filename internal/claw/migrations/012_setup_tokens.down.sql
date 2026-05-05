-- DOWN
-- Rollback for 012_setup_tokens.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP TABLE IF EXISTS np_claw.setup_tokens CASCADE;
-- DROP SCHEMA IF EXISTS np_claw CASCADE;  -- uncomment only if schema is exclusively owned by this migration
