-- Rollback: P93 S31-T01 RLS audit extension.
-- Drops audit artifacts ONLY. Does NOT disable RLS on tables — that would be a security regression.
-- To fully roll back RLS on a specific table, ALTER TABLE <t> DISABLE ROW LEVEL SECURITY manually.

BEGIN;

DROP VIEW IF EXISTS np_claw_rls_coverage;
DROP TABLE IF EXISTS np_claw_rls_audit;
DROP TABLE IF EXISTS np_claw_rls_shared_lookup;

COMMIT;
