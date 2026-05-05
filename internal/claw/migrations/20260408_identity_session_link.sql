-- T02.5: Link sessions to identities (personas).
-- Tracks which persona was active when a session was created.
ALTER TABLE np_claw_sessions ADD COLUMN IF NOT EXISTS identity_id UUID REFERENCES np_claw_personas(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_identity_id ON np_claw_sessions(identity_id) WHERE identity_id IS NOT NULL;
