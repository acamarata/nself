-- P93 S31-T04: Soft-delete pattern on 4 high-value claw tables.
-- Tables: np_claw_memories, np_claw_sessions, np_claw_projects, np_claw_topics
-- Rollback: 20260417_s31_softdelete_core_rollback.sql

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
            EXECUTE format('ALTER TABLE %I ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ', t);
            EXECUTE format(
                'CREATE INDEX IF NOT EXISTS idx_%s_not_deleted ON %I (id) WHERE deleted_at IS NULL',
                t, t
            );
            EXECUTE format(
                'CREATE INDEX IF NOT EXISTS idx_%s_deleted_at ON %I (deleted_at) WHERE deleted_at IS NOT NULL',
                t, t
            );
            EXECUTE format(
                'CREATE OR REPLACE VIEW %s_active AS SELECT * FROM %I WHERE deleted_at IS NULL',
                t, t
            );
        END IF;
    END LOOP;
END $$;

COMMIT;
