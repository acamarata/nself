-- One-time cleanup: delete empty test sessions (no messages).
-- These were created during development/testing and have no user value.

DELETE FROM np_claw_sessions
WHERE id NOT IN (
  SELECT DISTINCT session_id FROM np_claw_messages
)
AND created_at < NOW() - INTERVAL '1 hour';
