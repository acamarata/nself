-- DOWN
-- Rollback for 20260414_p86_s09_t09_04_memory_rooms_topic_unique.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS uq_memory_rooms_user_topic;
-- Unrecognized statement (no auto-inverse): WITH ranked AS (...
-- Unrecognized statement (no auto-inverse): WITH ranked AS (...
-- update np_claw_memory_rooms r
set topic_id = t.id
from np_cl... cannot be auto-reversed; verify manually if rollback needed.
DROP INDEX IF EXISTS idx_memory_rooms_topic;
ALTER TABLE IF EXISTS np_claw_memory_rooms DROP COLUMN IF EXISTS topic_id;
