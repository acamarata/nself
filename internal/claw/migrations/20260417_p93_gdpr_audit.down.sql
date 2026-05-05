-- DOWN
-- Rollback for 20260417_p93_gdpr_audit.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_gdpr_delete_requests_user;
DROP TABLE IF EXISTS np_claw_gdpr_delete_requests CASCADE;
DROP INDEX IF EXISTS idx_np_claw_gdpr_export_jobs_user;
DROP TABLE IF EXISTS np_claw_gdpr_export_jobs CASCADE;
DROP INDEX IF EXISTS idx_np_claw_audit_trail_action;
DROP INDEX IF EXISTS idx_np_claw_audit_trail_user;
DROP TABLE IF EXISTS np_claw_audit_trail CASCADE;
