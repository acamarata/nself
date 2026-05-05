-- DOWN
-- Rollback for 007_cards.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_claw_cards_namespace;
DROP INDEX IF EXISTS idx_claw_cards_user_unread;
DROP TABLE IF EXISTS np_claw_cards CASCADE;
