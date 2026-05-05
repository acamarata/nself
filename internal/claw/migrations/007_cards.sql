-- np_claw_cards: companion notification cards persisted from mux CompanionNotify action
-- T-0826

CREATE TABLE IF NOT EXISTS np_claw_cards (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT,
    namespace   TEXT        NOT NULL DEFAULT 'default',
    card_type   TEXT        NOT NULL DEFAULT 'notification',
    title       TEXT        NOT NULL,
    body        TEXT        NOT NULL DEFAULT '',
    priority    INTEGER     NOT NULL DEFAULT 0,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_claw_cards_user_unread
    ON np_claw_cards(user_id, created_at DESC)
    WHERE read_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_claw_cards_namespace
    ON np_claw_cards(namespace);
