-- P87 Sprint 01, Ticket 01.T10: Message Summaries Table (Compression Cache)
-- Spec §4.10

-- UP

CREATE TABLE IF NOT EXISTS np_claw_message_summaries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL,
    tier            INT NOT NULL CHECK (tier >= 2 AND tier <= 4),
    range_start     INT NOT NULL,
    range_end       INT NOT NULL,
    summary_text    TEXT NOT NULL,
    token_count     INT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_summaries_session_tier_range UNIQUE (session_id, tier, range_start, range_end)
);

-- DOWN
-- DROP TABLE IF EXISTS np_claw_message_summaries;
