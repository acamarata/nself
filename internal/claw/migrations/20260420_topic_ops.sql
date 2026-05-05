-- P87 Sprint 01, Ticket 01.T06: Topic Operations + Lifecycle Columns
-- Spec §4.6

-- UP

ALTER TABLE np_claw_topics ADD COLUMN IF NOT EXISTS archived_at     TIMESTAMPTZ;
ALTER TABLE np_claw_topics ADD COLUMN IF NOT EXISTS last_active_at  TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE np_claw_topics ADD COLUMN IF NOT EXISTS merged_into     UUID REFERENCES np_claw_topics(id) ON DELETE SET NULL;
ALTER TABLE np_claw_topics ADD COLUMN IF NOT EXISTS split_from      UUID REFERENCES np_claw_topics(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS np_claw_topic_ops_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    op              TEXT NOT NULL CHECK (op IN ('merge','split','reparent','archive','unarchive','rename')),
    src_topic_id    UUID REFERENCES np_claw_topics(id) ON DELETE SET NULL,
    dst_topic_id    UUID REFERENCES np_claw_topics(id) ON DELETE SET NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    performed_by    TEXT NOT NULL DEFAULT 'user' CHECK (performed_by IN ('user','agent','system')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE np_claw_topic_ops_log ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_topic_ops_log' AND policyname = 'topic_ops_user_isolation') THEN
        CREATE POLICY topic_ops_user_isolation ON np_claw_topic_ops_log
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

-- DOWN
-- DROP POLICY IF EXISTS topic_ops_user_isolation ON np_claw_topic_ops_log;
-- DROP TABLE IF EXISTS np_claw_topic_ops_log;
-- ALTER TABLE np_claw_topics DROP COLUMN IF EXISTS split_from;
-- ALTER TABLE np_claw_topics DROP COLUMN IF EXISTS merged_into;
-- ALTER TABLE np_claw_topics DROP COLUMN IF EXISTS last_active_at;
-- ALTER TABLE np_claw_topics DROP COLUMN IF EXISTS archived_at;
