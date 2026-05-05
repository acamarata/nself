-- DOWN
-- Rollback for 20260413_memory_is_global_default.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- update np_claw_memories
set is_global = true
where topic_id ... cannot be auto-reversed; verify manually if rollback needed.
