-- DOWN
-- Rollback for 20260413_p86_sprint08.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_share_comments_share;
DROP TABLE IF EXISTS np_claw_share_comments CASCADE;
-- update np_claw_proactive_jobs
set next_run_at = now() + inte... cannot be auto-reversed; verify manually if rollback needed.
-- delete from np_claw_topics
where message_count = 0
  and nam... cannot be auto-reversed; verify manually if rollback needed.
