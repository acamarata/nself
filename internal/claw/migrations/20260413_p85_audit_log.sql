-- Phase 85 Sprint 10 T02: Admin audit log
-- Tracks all admin actions (user disable, role change, password reset, etc.)
-- for security review and compliance.

CREATE TABLE IF NOT EXISTS np_claw_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id TEXT NOT NULL,
    actor_role TEXT NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT,
    target_id TEXT,
    details JSONB,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_actor ON np_claw_audit_log (actor_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON np_claw_audit_log (action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON np_claw_audit_log (created_at DESC);
