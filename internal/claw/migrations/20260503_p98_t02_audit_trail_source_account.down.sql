-- P98 S09-T02 rollback: remove source_account_id from np_claw_audit_trail.
--
-- NOTE: DROP INDEX CONCURRENTLY cannot run inside a transaction block.
-- Run it separately before the ALTER TABLE.

DROP INDEX CONCURRENTLY IF EXISTS idx_np_claw_audit_trail_source_account;
ALTER TABLE np_claw_audit_trail DROP COLUMN IF EXISTS source_account_id;
