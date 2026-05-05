-- DOWN
-- Rollback for 014_repo_monitors.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_claw_monitor_events_relevance;
DROP INDEX IF EXISTS idx_claw_monitor_events_monitor;
DROP INDEX IF EXISTS idx_claw_monitors_enabled;
DROP TABLE IF EXISTS np_claw_monitor_events CASCADE;
DROP TABLE IF EXISTS np_claw_monitors CASCADE;
