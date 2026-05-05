-- DOWN
-- Rollback for 20260418_p94_rls_completion.sql
-- Drops the user_isolation / thread_isolation policies and disables RLS on the
-- 14 tables covered by P94 completion sweep. Leaves policies created by
-- earlier migrations untouched.
-- Idempotent.

DO $$
DECLARE
    t TEXT;
    tables TEXT[] := ARRAY[
        'cl_threads',
        'np_ai_free_tier_quota',
        'np_ai_recommendation_actioned',
        'np_auth_devices',
        'np_auth_mfa',
        'np_auth_refresh_tokens',
        'np_claw_audit_trail',
        'np_claw_conversations',
        'np_claw_gdpr_delete_requests',
        'np_claw_gdpr_export_jobs',
        'np_notify_jobs'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = t) THEN
            BEGIN
                EXECUTE format('DROP POLICY IF EXISTS user_isolation ON %I', t);
                EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
                EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', t);
            EXCEPTION WHEN OTHERS THEN
                RAISE WARNING 'P94 RLS rollback: skipped table % — %: %', t, SQLSTATE, SQLERRM;
            END;
        END IF;
    END LOOP;
END $$;

DO $$
DECLARE
    t TEXT;
    tables TEXT[] := ARRAY[
        'cl_messages',
        'cl_memory_blocks',
        'cl_thread_cores'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = t) THEN
            BEGIN
                EXECUTE format('DROP POLICY IF EXISTS thread_isolation ON %I', t);
                EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
                EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', t);
            EXCEPTION WHEN OTHERS THEN
                RAISE WARNING 'P94 RLS rollback (transitive): skipped table % — %: %', t, SQLSTATE, SQLERRM;
            END;
        END IF;
    END LOOP;
END $$;
