-- P98 S09-T02: Add source_account_id to np_claw_audit_trail for multi-app isolation.
-- Per PPI multi-tenant convention wall: source_account_id separates independent consumer
-- apps within one nSelf deploy. Previously missing from this table.
--
-- NOTE: The CREATE INDEX CONCURRENTLY step must run during a low-traffic window.
-- It cannot run inside a transaction block. If applying via a migration framework
-- that wraps DDL in transactions, split this file into two steps:
--   1. ALTER TABLE (safe inside a transaction)
--   2. CREATE INDEX CONCURRENTLY (run separately, outside any transaction)

ALTER TABLE np_claw_audit_trail
    ADD COLUMN IF NOT EXISTS source_account_id TEXT NOT NULL DEFAULT 'primary';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_np_claw_audit_trail_source_account
    ON np_claw_audit_trail(source_account_id, user_id, created_at DESC);
