-- P92 S10-02: Row-Level Security — tenant isolation on 10 sensitive tables.
-- Any query on these tables must first set `SET LOCAL app.user_id = '<uuid>'`.
-- FORCE ROW LEVEL SECURITY ensures the policy applies even to table owners.

DO $$
DECLARE
    t TEXT;
    tables TEXT[] := ARRAY[
        'np_claw_memories',
        'np_claw_sessions',
        'np_claw_threads',
        'np_claw_messages',
        'np_claw_oauth_tokens',
        'np_claw_audit_log',
        'np_claw_cost_events',
        'np_claw_touchpoints',
        'np_claw_skills',
        'np_claw_plan_revenue'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = t) THEN
            EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
            EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);

            -- Drop existing policy if present (idempotent).
            EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);

            EXECUTE format(
                'CREATE POLICY tenant_isolation ON %I '
                'USING (user_id = NULLIF(current_setting(''app.user_id'', true), '''')::uuid)',
                t
            );
        END IF;
    END LOOP;
END $$;
