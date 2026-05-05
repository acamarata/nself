-- P87 Sprint 01, Ticket 01.T04: User Vocabulary Table
-- Spec §4.4

-- UP

CREATE TABLE IF NOT EXISTS np_claw_user_vocabulary (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL,
    term                TEXT NOT NULL,
    normalized_term     TEXT GENERATED ALWAYS AS (lower(term)) STORED,
    expansion           TEXT NOT NULL,
    category            TEXT CHECK (category IN ('acronym','nickname','project','person','jargon','alias')),
    confidence          REAL DEFAULT 0.6 CHECK (confidence >= 0 AND confidence <= 1),
    usage_count         INT NOT NULL DEFAULT 1,
    last_used_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    source_message_id   UUID REFERENCES np_claw_messages(id) ON DELETE SET NULL,
    user_confirmed      BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_vocab_user_term UNIQUE (user_id, normalized_term)
);

CREATE INDEX IF NOT EXISTS idx_vocab_user_conf
    ON np_claw_user_vocabulary(user_id, confidence DESC, usage_count DESC);

ALTER TABLE np_claw_user_vocabulary ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_user_vocabulary' AND policyname = 'vocab_user_isolation') THEN
        CREATE POLICY vocab_user_isolation ON np_claw_user_vocabulary
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

-- DOWN
-- DROP POLICY IF EXISTS vocab_user_isolation ON np_claw_user_vocabulary;
-- DROP TABLE IF EXISTS np_claw_user_vocabulary;
