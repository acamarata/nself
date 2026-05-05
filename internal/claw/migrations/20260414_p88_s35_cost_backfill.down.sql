-- DOWN
-- Rollback for 20260414_p88_s35_cost_backfill.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- delete from np_cost_rollup_cursor where job_name in ('ai_cos... cannot be auto-reversed; verify manually if rollback needed.
-- Unrecognized statement (no auto-inverse): WITH resolved AS (...
DROP TABLE IF EXISTS np_ai_cost_reconciliation_log CASCADE;
DROP INDEX IF EXISTS idx_cron_run_log_job_name;
DROP TABLE IF EXISTS np_cron_run_log CASCADE;
DROP TABLE IF EXISTS np_cost_rollup_cursor CASCADE;
