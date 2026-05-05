-- Migration: np_claw_agent_tasks
-- Creates the agent tasks table used by sprint15_handlers.go

CREATE TABLE IF NOT EXISTS np_claw_agent_tasks (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id     UUID        NOT NULL REFERENCES np_claw_agents(id) ON DELETE CASCADE,
    task         TEXT        NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    result       JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_claw_agent_tasks_agent_id ON np_claw_agent_tasks(agent_id);
CREATE INDEX IF NOT EXISTS idx_claw_agent_tasks_status   ON np_claw_agent_tasks(status);
