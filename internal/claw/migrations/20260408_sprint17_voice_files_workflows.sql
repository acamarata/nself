-- Sprint 17: Voice persona, project files, workflow templates enhancements

-- T17.1: Add voice_id to identities for persona-specific voice selection
ALTER TABLE np_claw_identities ADD COLUMN IF NOT EXISTS voice_id TEXT;
ALTER TABLE np_claw_identities ADD COLUMN IF NOT EXISTS voice_provider TEXT;

-- T17.4: Project files table (if not exists from S12 scaffold)
CREATE TABLE IF NOT EXISTS np_claw_project_files (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID        NOT NULL REFERENCES np_claw_projects(id) ON DELETE CASCADE,
    user_id      TEXT        NOT NULL DEFAULT 'anonymous',
    filename     TEXT        NOT NULL,
    mimetype     TEXT        NOT NULL DEFAULT 'application/octet-stream',
    size         INTEGER     NOT NULL DEFAULT 0,
    storage_path TEXT        NOT NULL,
    version      INTEGER     NOT NULL DEFAULT 1,
    parent_id    UUID        REFERENCES np_claw_project_files(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_project_files_project ON np_claw_project_files(project_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_project_files_parent ON np_claw_project_files(parent_id);

-- T17.6: Add sharing columns to workflow templates
ALTER TABLE np_claw_workflow_templates ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE np_claw_workflow_templates ADD COLUMN IF NOT EXISTS share_token TEXT;
ALTER TABLE np_claw_workflow_templates ADD COLUMN IF NOT EXISTS variables JSONB DEFAULT '[]'::jsonb;
ALTER TABLE np_claw_workflow_templates ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE np_claw_workflow_templates ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_np_claw_workflow_templates_share ON np_claw_workflow_templates(share_token) WHERE share_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_np_claw_workflow_templates_public ON np_claw_workflow_templates(is_public) WHERE is_public = true;

-- T17.3: Session search improvements (add identity_id, persona for filtering)
-- Already have full-text index from 20260403_message_fulltext.sql
-- Add persona filter index
CREATE INDEX IF NOT EXISTS idx_np_claw_sessions_identity ON np_claw_sessions(identity_id) WHERE identity_id IS NOT NULL;
