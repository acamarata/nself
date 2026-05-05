-- T-0950: Persistent agent queue for nself-claw → nClaw companion dispatch.
--
-- Actions dispatched by the nself-claw Rust plugin are stored here so they
-- survive companion disconnections and are delivered when the companion
-- reconnects (drain-on-connect protocol).

CREATE TABLE IF NOT EXISTS np_claw_agent_queue (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace        TEXT         NOT NULL,           -- user namespace (maps to companion app)
    action           TEXT         NOT NULL,           -- action type: "oauth_reauth_required", "file_op", "shell_exec", "browser_auth", "notification", etc.
    payload          JSONB        NOT NULL DEFAULT '{}'::jsonb,
    idempotency_key  TEXT,                            -- caller-provided idempotency key (unique per namespace)
    status           TEXT         NOT NULL DEFAULT 'pending',  -- pending, dispatched, completed, failed, expired
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at       TIMESTAMPTZ  NOT NULL DEFAULT (now() + INTERVAL '24 hours'),
    dispatched_at    TIMESTAMPTZ,                     -- when first sent to companion
    completed_at     TIMESTAMPTZ,                     -- when companion acknowledged
    result           JSONB,                           -- companion result payload
    attempt_count    INTEGER      NOT NULL DEFAULT 0,
    CONSTRAINT np_claw_agent_queue_idempotency UNIQUE (namespace, idempotency_key)
        DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_np_claw_agent_queue_namespace_status
    ON np_claw_agent_queue (namespace, status, created_at);

CREATE INDEX IF NOT EXISTS idx_np_claw_agent_queue_expires
    ON np_claw_agent_queue (expires_at)
    WHERE status IN ('pending', 'dispatched');
