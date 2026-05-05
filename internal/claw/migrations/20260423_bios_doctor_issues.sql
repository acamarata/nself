-- Migration: np_claw_doctor_issues (SP-20 C17b Layer 4)
-- Persists unfixable BIOS drift events for admin UI + `nself claw doctor` surface.

CREATE TABLE IF NOT EXISTS np_claw_doctor_issues (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  issue_code   TEXT NOT NULL,
  message      TEXT NOT NULL,
  detail       JSONB,
  severity     TEXT NOT NULL DEFAULT 'critical' CHECK (severity IN ('info','warning','critical')),
  resolved     BOOLEAN NOT NULL DEFAULT FALSE,
  resolved_at  TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_doctor_issues_unresolved
  ON np_claw_doctor_issues (resolved, created_at DESC)
  WHERE resolved = FALSE;

CREATE INDEX IF NOT EXISTS idx_doctor_issues_code
  ON np_claw_doctor_issues (issue_code, created_at DESC);

-- Hasura Row-Level Security note:
--   np_claw_doctor_issues: admin-only access (no tenant filter — deploy-global)
--   Hasura role filter: {"_exists": {"_table": {"name": "nself_accounts"}, "_where": {"is_admin": {"_eq": true}}}}
