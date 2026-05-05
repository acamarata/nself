-- AA02 rollback

BEGIN;

DROP INDEX IF EXISTS idx_claw_agent_tasks_timeout;
DROP INDEX IF EXISTS idx_claw_agents_heartbeat;

ALTER TABLE np_claw_agent_tasks
  DROP COLUMN IF EXISTS timeout_at,
  DROP COLUMN IF EXISTS error_message,
  DROP COLUMN IF EXISTS fallback_used;

ALTER TABLE np_claw_agents
  DROP COLUMN IF EXISTS fallback_model,
  DROP COLUMN IF EXISTS consecutive_degraded,
  DROP COLUMN IF EXISTS last_heartbeat,
  DROP COLUMN IF EXISTS status;

COMMIT;
