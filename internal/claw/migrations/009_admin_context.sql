-- np_claw_admin_context: stack state snapshot storage for admin context engine
-- T-1082

CREATE TABLE IF NOT EXISTS np_claw_admin_context (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  snapshot_type TEXT        NOT NULL DEFAULT 'stack_state', -- 'stack_state' | 'schema' | 'custom'
  payload       JSONB       NOT NULL,
  captured_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_current    BOOLEAN     NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_np_claw_admin_context_current
    ON np_claw_admin_context(snapshot_type, is_current);
