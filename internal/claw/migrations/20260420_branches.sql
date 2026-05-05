-- P87 Sprint 01, Ticket 01.T12: Branches Table
-- Spec §4.12

-- UP

CREATE TABLE IF NOT EXISTS np_claw_branches (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL,
    session_id          UUID NOT NULL,
    parent_branch_id    UUID REFERENCES np_claw_branches(id) ON DELETE SET NULL,
    fork_message_id     UUID REFERENCES np_claw_messages(id) ON DELETE SET NULL,
    name                TEXT,
    merged_into         UUID REFERENCES np_claw_branches(id) ON DELETE SET NULL,
    merged_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_branches_session ON np_claw_branches(session_id);

ALTER TABLE np_claw_branches ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_branches' AND policyname = 'branches_user_isolation') THEN
        CREATE POLICY branches_user_isolation ON np_claw_branches
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

-- DOWN
-- DROP POLICY IF EXISTS branches_user_isolation ON np_claw_branches;
-- DROP TABLE IF EXISTS np_claw_branches;
