-- AA02: agent executor production gaps
-- Adds: timeout_at, error_message, fallback_used to np_claw_agent_tasks
--       fallback_model, consecutive_degraded, last_heartbeat, status to np_claw_agents

BEGIN;

ALTER TABLE np_claw_agent_tasks
  ADD COLUMN IF NOT EXISTS timeout_at    timestamptz,
  ADD COLUMN IF NOT EXISTS error_message text,
  ADD COLUMN IF NOT EXISTS fallback_used boolean NOT NULL DEFAULT false;

ALTER TABLE np_claw_agents
  ADD COLUMN IF NOT EXISTS fallback_model       text,
  ADD COLUMN IF NOT EXISTS consecutive_degraded int  NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_heartbeat       timestamptz,
  ADD COLUMN IF NOT EXISTS status               text NOT NULL DEFAULT 'idle';

-- Index to make the watchdog sweep fast
CREATE INDEX IF NOT EXISTS idx_claw_agent_tasks_timeout
  ON np_claw_agent_tasks (status, timeout_at)
  WHERE status = 'running' AND timeout_at IS NOT NULL;

-- Index to make the stale-agent sweep fast
CREATE INDEX IF NOT EXISTS idx_claw_agents_heartbeat
  ON np_claw_agents (status, last_heartbeat)
  WHERE status = 'running';

COMMIT;
