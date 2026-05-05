-- DOWN
-- Rollback for 20260407_pending_approvals.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_pending_approvals_chat;
DROP INDEX IF EXISTS idx_pending_approvals_status;
DROP TABLE IF EXISTS np_claw.pending_approvals CASCADE;
