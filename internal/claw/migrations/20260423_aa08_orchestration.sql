-- AA08: Claw plugin orchestration engine — capability registry + run state tables.

CREATE TABLE IF NOT EXISTS np_claw_capabilities (
    id                    uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin                text        NOT NULL,
    action                text        NOT NULL,
    description           text,
    input_schema          jsonb,
    output_schema         jsonb,
    requires_confirmation boolean     NOT NULL DEFAULT false,
    tags                  text[]      NOT NULL DEFAULT '{}',
    UNIQUE (plugin, action)
);

CREATE INDEX IF NOT EXISTS idx_np_claw_capabilities_plugin
    ON np_claw_capabilities (plugin);

CREATE INDEX IF NOT EXISTS idx_np_claw_capabilities_tags
    ON np_claw_capabilities USING gin (tags);

CREATE TABLE IF NOT EXISTS np_claw_orchestration_runs (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id uuid,
    message_id      uuid,
    plan            jsonb       NOT NULL,
    status          text        NOT NULL DEFAULT 'planning'
                                CHECK (status IN (
                                    'planning', 'running', 'waiting_confirmation',
                                    'done', 'error', 'rolled_back'
                                )),
    current_step    int         NOT NULL DEFAULT 0,
    context         jsonb       NOT NULL DEFAULT '{}',
    error           text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_orch_runs_status
    ON np_claw_orchestration_runs (status)
    WHERE status IN ('running', 'planning', 'waiting_confirmation');

CREATE INDEX IF NOT EXISTS idx_np_claw_orch_runs_conversation
    ON np_claw_orchestration_runs (conversation_id);
