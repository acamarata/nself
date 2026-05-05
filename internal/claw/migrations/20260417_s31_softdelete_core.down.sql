-- Down migration for 20260417_s31_softdelete_core.sql
-- Removes deleted_at columns, partial indexes, and _active views
-- from np_claw_memories, np_claw_sessions, np_claw_projects, np_claw_topics.

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
        IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = t) THEN
            -- Drop _active view
            EXECUTE format('DROP VIEW IF EXISTS %s_active', t);
            -- Drop partial indexes on deleted_at
            EXECUTE format('DROP INDEX IF EXISTS idx_%s_deleted_at', t);
            EXECUTE format('DROP INDEX IF EXISTS idx_%s_not_deleted', t);
            -- Remove deleted_at column
            EXECUTE format('ALTER TABLE %I DROP COLUMN IF EXISTS deleted_at', t);
        END IF;
    END LOOP;
END $$;

COMMIT;
