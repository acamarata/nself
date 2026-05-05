-- P94 RLS completion: enable Row-Level Security on remaining user-scoped tables.
--
-- DEEP-QA finding: only 16/99 claw migrations have ENABLE ROW LEVEL SECURITY.
-- Tables created between S31 and P94 (audit trail, GDPR jobs, conversations,
-- auth devices/MFA/refresh tokens, notify jobs, v2 memory tables, AI free-tier
-- quota) were omitted from the earlier 20260417_p93_rls_all_tables.sql sweep.
--
-- Pattern: user_id = NULLIF(current_setting('app.user_id', true), '')::uuid
-- Transitive tables (cl_messages, cl_memory_blocks, cl_thread_cores) scope via
-- their thread_id FK into cl_threads.
--
-- Idempotent: safe to re-run. Uses IF EXISTS + DROP POLICY IF EXISTS.
-- Rollback: see 20260418_p94_rls_completion.down.sql

-- ─────────────────────────────────────────────
-- 1. Tables with a direct user_id column
-- ─────────────────────────────────────────────

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
                EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
                EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
                EXECUTE format('DROP POLICY IF EXISTS user_isolation ON %I', t);
                EXECUTE format(
                    'CREATE POLICY user_isolation ON %I '
                    'USING (user_id = NULLIF(current_setting(''app.user_id'', true), '''')::uuid) '
                    'WITH CHECK (user_id = NULLIF(current_setting(''app.user_id'', true), '''')::uuid)',
                    t
                );
            EXCEPTION WHEN OTHERS THEN
                RAISE WARNING 'P94 RLS: skipped table % — %: %', t, SQLSTATE, SQLERRM;
            END;
        END IF;
    END LOOP;
END $$;

-- ─────────────────────────────────────────────
-- 2. Transitively user-scoped cl_* tables (thread_id → cl_threads.user_id)
-- ─────────────────────────────────────────────

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
                EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
                EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
                EXECUTE format('DROP POLICY IF EXISTS thread_isolation ON %I', t);
                EXECUTE format(
                    'CREATE POLICY thread_isolation ON %I '
                    'USING (thread_id IN (SELECT id FROM cl_threads WHERE user_id = NULLIF(current_setting(''app.user_id'', true), '''')::uuid)) '
                    'WITH CHECK (thread_id IN (SELECT id FROM cl_threads WHERE user_id = NULLIF(current_setting(''app.user_id'', true), '''')::uuid))',
                    t
                );
            EXCEPTION WHEN OTHERS THEN
                RAISE WARNING 'P94 RLS (transitive): skipped table % — %: %', t, SQLSTATE, SQLERRM;
            END;
        END IF;
    END LOOP;
END $$;
