-- nself-claw: admin mode flag on sessions (T-1085)
-- Idempotent — IF NOT EXISTS / ADD COLUMN IF NOT EXISTS

ALTER TABLE np_claw_sessions
    ADD COLUMN IF NOT EXISTS is_admin_mode BOOLEAN NOT NULL DEFAULT false;
