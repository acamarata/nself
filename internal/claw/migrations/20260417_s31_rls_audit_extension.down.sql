-- Down migration for 20260417_s31_rls_audit_extension.sql
-- Drops the RLS audit infrastructure: coverage view, shared-lookup table, audit table.
-- Also disables RLS / drops policies added by the DO block on discovered tables.

BEGIN;

-- Drop coverage view
DROP VIEW IF EXISTS np_claw_rls_coverage;

-- Disable RLS and drop policies on all np_claw_* tables that were covered
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT tablename FROM pg_tables
        WHERE schemaname = 'public'
          AND tablename LIKE 'np\_claw\_%' ESCAPE '\'
          AND tablename NOT IN ('np_claw_rls_audit', 'np_claw_rls_shared_lookup')
        ORDER BY tablename
    LOOP
        BEGIN
            EXECUTE format('DROP POLICY IF EXISTS user_isolation ON %I', r.tablename);
            EXECUTE format('DROP POLICY IF EXISTS shared_readonly ON %I', r.tablename);
            EXECUTE format('DROP POLICY IF EXISTS deny_all ON %I', r.tablename);
            EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', r.tablename);
            EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', r.tablename);
        EXCEPTION WHEN OTHERS THEN
            RAISE WARNING 'S31-T01 rollback: skipped table % — %: %', r.tablename, SQLSTATE, SQLERRM;
        END;
    END LOOP;
END $$;

-- Drop audit infrastructure tables
DROP TABLE IF EXISTS np_claw_rls_shared_lookup;
DROP TABLE IF EXISTS np_claw_rls_audit;

COMMIT;
