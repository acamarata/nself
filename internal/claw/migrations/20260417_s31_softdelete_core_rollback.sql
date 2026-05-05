-- Rollback: P93 S31-T04 soft-delete on 4 core tables.
-- Drops the _active views + deleted_at columns. Does NOT restore deleted rows.

BEGIN;

DO $$
DECLARE
    t TEXT;
    tables TEXT[] := ARRAY[
        'np_claw_memories',
        'np_claw_sessions',
        'np_claw_projects',
        'np_claw_topics'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP VIEW IF EXISTS %s_active', t);
        EXECUTE format('DROP INDEX IF EXISTS idx_%s_deleted_at', t);
        EXECUTE format('DROP INDEX IF EXISTS idx_%s_not_deleted', t);
        IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = t) THEN
            EXECUTE format('ALTER TABLE %I DROP COLUMN IF EXISTS deleted_at', t);
        END IF;
    END LOOP;
END $$;

COMMIT;
