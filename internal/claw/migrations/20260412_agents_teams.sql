-- Bug fix: np_claw_agents, np_claw_teams, np_claw_team_members were referenced in
-- Go code (agent/agents.go, tools/agents.go, server/sprint15_handlers.go) but no
-- migration created them. This file adds all three tables idempotently.

-- Agents: autonomous task executors with configurable model, memory scope, and budget
CREATE TABLE IF NOT EXISTS np_claw_agents (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID,
    name                 TEXT        NOT NULL,
    template             TEXT,
    system_prompt        TEXT,
    model_preference     TEXT,
    tier_ceiling         INT         NOT NULL DEFAULT 3,
    memory_scope         TEXT        NOT NULL DEFAULT 'isolated',
    budget_daily_usd     DOUBLE PRECISION,
    tools_allowed        TEXT[]      NOT NULL DEFAULT '{}',
    enabled              BOOLEAN     NOT NULL DEFAULT true,
    schedule_cron        TEXT,
    consecutive_failures INT         NOT NULL DEFAULT 0,
    last_successful_task TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_agents_user ON np_claw_agents(user_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_agents_enabled ON np_claw_agents(enabled) WHERE enabled = true;

-- Teams: named groups of agents with an optional master coordinator agent
CREATE TABLE IF NOT EXISTS np_claw_teams (
    id              UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT    NOT NULL,
    description     TEXT,
    master_agent_id UUID    REFERENCES np_claw_agents(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Team membership: many-to-many between agents and teams with a role
CREATE TABLE IF NOT EXISTS np_claw_team_members (
    team_id     UUID    NOT NULL REFERENCES np_claw_teams(id) ON DELETE CASCADE,
    agent_id    UUID    NOT NULL REFERENCES np_claw_agents(id) ON DELETE CASCADE,
    role        TEXT    NOT NULL DEFAULT 'member',
    PRIMARY KEY (team_id, agent_id)
);

CREATE INDEX IF NOT EXISTS idx_np_claw_team_members_agent ON np_claw_team_members(agent_id);
