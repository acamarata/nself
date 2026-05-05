-- Phase 86 Sprint 08 migrations.
-- T13: Clean up test topics with zero messages.
-- T14: Reset knowledge_distillation schedule.
-- T18: Shared conversation comments table.

-- T13: Remove test topics created during development/testing.
DELETE FROM np_claw_topics
WHERE message_count = 0
  AND name IN ('Test Topic', 'CRD Test Topic', 'Test');

-- T14: Reset knowledge_distillation next_run to 1 hour from now.
UPDATE np_claw_proactive_jobs
SET next_run_at = NOW() + INTERVAL '1 hour'
WHERE job_type = 'knowledge_distillation';

-- T18: Comments on shared conversations.
CREATE TABLE IF NOT EXISTS np_claw_share_comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    share_id    UUID NOT NULL REFERENCES np_claw_shared_sessions(id) ON DELETE CASCADE,
    author_name TEXT NOT NULL DEFAULT 'Anonymous',
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_share_comments_share
    ON np_claw_share_comments (share_id, created_at);
