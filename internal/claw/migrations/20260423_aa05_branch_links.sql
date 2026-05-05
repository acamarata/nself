-- AA05 — Branch/Merge Threads: Knowledge Graph Links + Pending Signals
-- Spec: AA05 (P95)

-- np_claw_branch_links stores cross-topic knowledge graph edges between branches.
CREATE TABLE IF NOT EXISTS np_claw_branch_links (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_id    UUID NOT NULL REFERENCES np_claw_branches(id) ON DELETE CASCADE,
    to_id      UUID NOT NULL REFERENCES np_claw_branches(id) ON DELETE CASCADE,
    link_type  TEXT NOT NULL CHECK(link_type IN ('related', 'contradicts', 'refines')),
    score      FLOAT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_branch_links_pair UNIQUE (from_id, to_id)
);

CREATE INDEX IF NOT EXISTS idx_branch_links_from ON np_claw_branch_links(from_id);
CREATE INDEX IF NOT EXISTS idx_branch_links_to   ON np_claw_branch_links(to_id);

ALTER TABLE np_claw_branch_links ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'np_claw_branch_links' AND policyname = 'branch_links_user_isolation'
    ) THEN
        -- Users can only see links for branches they own.
        CREATE POLICY branch_links_user_isolation ON np_claw_branch_links
            USING (
                from_id IN (
                    SELECT id FROM np_claw_branches
                    WHERE user_id = current_setting('hasura.user', true)::uuid
                )
            );
    END IF;
END $$;

-- np_claw_branch_pending_signals tracks first-crossing topic-switch signals.
-- A second crossing within 5 minutes confirms an auto-branch.
CREATE TABLE IF NOT EXISTS np_claw_branch_pending_signals (
    session_id         UUID NOT NULL,
    branch_id          UUID,  -- nullable = root branch
    trigger_message_id UUID NOT NULL,
    similarity         FLOAT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_pending_signals PRIMARY KEY (session_id, branch_id)
);

CREATE INDEX IF NOT EXISTS idx_pending_signals_session ON np_claw_branch_pending_signals(session_id);

-- Add centroid column to np_claw_branches (if not already present via earlier migration).
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'np_claw_branches' AND column_name = 'centroid'
    ) THEN
        ALTER TABLE np_claw_branches ADD COLUMN centroid vector(1536);
    END IF;
END $$;
