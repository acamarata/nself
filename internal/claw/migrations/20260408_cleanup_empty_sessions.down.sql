-- DOWN
-- Rollback for 20260408_cleanup_empty_sessions.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- delete from np_claw_sessions
where id not in (
  select dist... cannot be auto-reversed; verify manually if rollback needed.
