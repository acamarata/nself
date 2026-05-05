-- P93 S31-T01: RLS audit + extension — catch every user-scoped np_claw_* table.
-- Runs AFTER 20260420_rls_tenant_isolation.sql (P92) and 20260417_p93_rls_all_tables.sql (P93 S02).
--
-- Strategy:
--   1. Auto-discover every public table whose name matches np_claw_* or has a user_id uuid column.
--   2. For each: ENABLE + FORCE RLS, install user_isolation policy if not present.
--   3. Tables without user_id (shared lookup tables) get FORCE RLS + read-only-public policy.
--   4. Records audit trail in np_claw_rls_audit so ops can verify coverage.
--
-- Pattern for app code:
--   SET LOCAL app.user_id = '<uuid>';  -- required before any SELECT/INSERT/UPDATE/DELETE
--
-- Rollback: 20260417_s31_rls_audit_extension_rollback.sql

BEGIN;

-- Audit table: one row per np_claw_* table, tracks RLS state + coverage.
CREATE TABLE IF NOT EXISTS np_claw_rls_audit (
    table_name TEXT PRIMARY KEY,
    rls_enabled BOOLEAN NOT NULL,
    rls_forced BOOLEAN NOT NULL,
    has_user_id BOOLEAN NOT NULL,
    policy_name TEXT,
    audited_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Shared-lookup tables that are public-read, admin-write. Excluded from user_isolation policy.
-- If you add a shared lookup table, add it here.
CREATE TABLE IF NOT EXISTS np_claw_rls_shared_lookup (
    table_name TEXT PRIMARY KEY
);

INSERT INTO np_claw_rls_shared_lookup (table_name) VALUES
    ('np_claw_kg_predicates'),
    ('np_claw_tool_catalog_cache')
ON CONFLICT (table_name) DO NOTHING;

DO $$
DECLARE
    r RECORD;
    has_uid BOOLEAN;
    is_shared BOOLEAN;
    policy_exists BOOLEAN;
    policy_label TEXT;
BEGIN
    TRUNCATE np_claw_rls_audit;

    FOR r IN
        SELECT tablename FROM pg_tables
        WHERE schemaname = 'public'
          AND tablename LIKE 'np\_claw\_%' ESCAPE '\'
          AND tablename NOT IN ('np_claw_rls_audit', 'np_claw_rls_shared_lookup')
        ORDER BY tablename
    LOOP
        -- Check if table has user_id uuid column.
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = r.tablename
              AND column_name = 'user_id'
              AND data_type = 'uuid'
        ) INTO has_uid;

        SELECT EXISTS (
            SELECT 1 FROM np_claw_rls_shared_lookup WHERE table_name = r.tablename
        ) INTO is_shared;

        BEGIN
            EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', r.tablename);
            EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', r.tablename);

            IF is_shared THEN
                policy_label := 'shared_readonly';
                EXECUTE format('DROP POLICY IF EXISTS %I ON %I', policy_label, r.tablename);
                EXECUTE format(
                    'CREATE POLICY %I ON %I FOR SELECT USING (true)',
                    policy_label, r.tablename
                );
            ELSIF has_uid THEN
                policy_label := 'user_isolation';
                -- Don't overwrite existing user_isolation/tenant_isolation policies from prior migrations.
                SELECT EXISTS (
                    SELECT 1 FROM pg_policies
                    WHERE schemaname = 'public'
                      AND tablename = r.tablename
                      AND policyname IN ('user_isolation', 'tenant_isolation')
                ) INTO policy_exists;

                IF NOT policy_exists THEN
                    EXECUTE format(
                        'CREATE POLICY %I ON %I '
                        'USING (user_id = NULLIF(current_setting(''app.user_id'', true), '''')::uuid) '
                        'WITH CHECK (user_id = NULLIF(current_setting(''app.user_id'', true), '''')::uuid)',
                        policy_label, r.tablename
                    );
                ELSE
                    -- Reflect existing policy name in audit.
                    SELECT policyname INTO policy_label
                    FROM pg_policies
                    WHERE schemaname = 'public'
                      AND tablename = r.tablename
                      AND policyname IN ('user_isolation', 'tenant_isolation')
                    LIMIT 1;
                END IF;
            ELSE
                -- Table has no user_id and is not shared: deny-all is safest default.
                policy_label := 'deny_all';
                EXECUTE format('DROP POLICY IF EXISTS %I ON %I', policy_label, r.tablename);
                EXECUTE format(
                    'CREATE POLICY %I ON %I USING (false) WITH CHECK (false)',
                    policy_label, r.tablename
                );
                RAISE WARNING 'S31-T01: table % has no user_id — applied deny_all. Add to np_claw_rls_shared_lookup or add user_id column.', r.tablename;
            END IF;

            INSERT INTO np_claw_rls_audit (table_name, rls_enabled, rls_forced, has_user_id, policy_name)
            VALUES (r.tablename, true, true, has_uid, policy_label);
        EXCEPTION WHEN OTHERS THEN
            RAISE WARNING 'S31-T01: failed on table % — %: %', r.tablename, SQLSTATE, SQLERRM;
            INSERT INTO np_claw_rls_audit (table_name, rls_enabled, rls_forced, has_user_id, policy_name)
            VALUES (r.tablename, false, false, has_uid, NULL)
            ON CONFLICT (table_name) DO UPDATE SET
                rls_enabled = false,
                rls_forced = false,
                policy_name = NULL,
                audited_at = now();
        END;
    END LOOP;
END $$;

-- Coverage view: quick audit report.
CREATE OR REPLACE VIEW np_claw_rls_coverage AS
SELECT
    COUNT(*) AS total_tables,
    COUNT(*) FILTER (WHERE rls_enabled) AS rls_enabled_count,
    COUNT(*) FILTER (WHERE rls_forced) AS rls_forced_count,
    COUNT(*) FILTER (WHERE has_user_id) AS with_user_id_count,
    COUNT(*) FILTER (WHERE policy_name IS NOT NULL) AS with_policy_count,
    ROUND(100.0 * COUNT(*) FILTER (WHERE rls_enabled AND rls_forced AND policy_name IS NOT NULL) / NULLIF(COUNT(*), 0), 2) AS pct_fully_protected
FROM np_claw_rls_audit;

COMMIT;
