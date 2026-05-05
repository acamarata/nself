-- Pending approvals table for dangerous tool calls via Telegram.
-- When a tool marked as dangerous is invoked through the Telegram channel,
-- execution is paused until the user explicitly approves or rejects via
-- an inline keyboard button.

CREATE TABLE IF NOT EXISTS np_claw.pending_approvals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES np_claw.users(id) ON DELETE CASCADE,
    session_id  UUID,
    tg_chat_id  BIGINT NOT NULL,
    tool_name   TEXT NOT NULL,
    args        JSONB NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'expired')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pending_approvals_status ON np_claw.pending_approvals (status) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_pending_approvals_chat ON np_claw.pending_approvals (tg_chat_id, status);
