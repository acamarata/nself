-- DOWN
-- Rollback for 20260414_p86_t13_memory_rooms_unique.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_memory_rooms DROP CONSTRAINT IF EXISTS uq_memory_rooms_user_name;
-- delete from np_claw_memory_rooms
where id not in (
    selec... cannot be auto-reversed; verify manually if rollback needed.
