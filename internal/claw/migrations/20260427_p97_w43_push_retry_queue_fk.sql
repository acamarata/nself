-- P97 W43 — DB-M3 follow-up: add FK from np_claw_push_retry_queue.user_id
-- to np_users(id) with ON DELETE CASCADE.
--
-- Original migration (20260426_g2t04_push_retry_queue.sql) declared user_id
-- as TEXT NOT NULL with no FK. np_users.id is UUID. To create the FK we must:
--   1) Convert column type TEXT → UUID using ::uuid cast
--   2) Verify all existing rows are valid UUIDs that exist in np_users
--   3) Add FK with ON DELETE CASCADE
--
-- Safety guards:
--   - Wrapped in a transaction; rollback on any conversion failure
--   - Pre-flight check counts rows with non-UUID-shaped user_id; aborts if any
--   - Pre-flight check counts orphan rows (user_id not in np_users); aborts if any
--   - Idempotent: skips the work if FK constraint already exists
--
-- If pre-flight aborts (orphans / malformed rows), operator must reconcile
-- those rows manually before re-running. See:
--   .claude/qa/bugs/p97-w43-db-m3-orphan-cleanup.md

BEGIN;

DO $$
DECLARE
  v_fk_exists       BOOLEAN;
  v_col_type        TEXT;
  v_bad_uuid_count  INT;
  v_orphan_count    INT;
BEGIN
  -- Skip if FK already exists (idempotent re-run)
  SELECT EXISTS (
    SELECT 1
      FROM pg_constraint
     WHERE conname = 'np_claw_push_retry_queue_user_fkey'
  ) INTO v_fk_exists;

  IF v_fk_exists THEN
    RAISE NOTICE 'DB-M3: FK np_claw_push_retry_queue_user_fkey already exists; skipping.';
    RETURN;
  END IF;

  -- Discover current column type
  SELECT data_type
    FROM information_schema.columns
   WHERE table_schema = 'public'
     AND table_name   = 'np_claw_push_retry_queue'
     AND column_name  = 'user_id'
    INTO v_col_type;

  -- TEXT → UUID conversion if needed
  IF v_col_type = 'text' THEN
    -- Pre-flight: any rows with non-UUID-shaped user_id?
    EXECUTE $check$
      SELECT COUNT(*)
        FROM np_claw_push_retry_queue
       WHERE user_id !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    $check$ INTO v_bad_uuid_count;

    IF v_bad_uuid_count > 0 THEN
      RAISE EXCEPTION 'DB-M3 abort: % rows in np_claw_push_retry_queue have non-UUID user_id values. Reconcile manually before re-running.', v_bad_uuid_count;
    END IF;

    ALTER TABLE np_claw_push_retry_queue
      ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
  END IF;

  -- Pre-flight: orphan rows (user_id not in np_users)
  EXECUTE $check$
    SELECT COUNT(*)
      FROM np_claw_push_retry_queue q
     WHERE NOT EXISTS (SELECT 1 FROM np_users u WHERE u.id = q.user_id)
  $check$ INTO v_orphan_count;

  IF v_orphan_count > 0 THEN
    RAISE EXCEPTION 'DB-M3 abort: % rows in np_claw_push_retry_queue reference non-existent users. Reconcile manually (delete orphans or seed users) before re-running.', v_orphan_count;
  END IF;

  -- Add FK with cascade-on-delete
  EXECUTE $sql$
    ALTER TABLE np_claw_push_retry_queue
      ADD CONSTRAINT np_claw_push_retry_queue_user_fkey
      FOREIGN KEY (user_id) REFERENCES np_users(id) ON DELETE CASCADE
  $sql$;

  RAISE NOTICE 'DB-M3: FK np_claw_push_retry_queue_user_fkey added (ON DELETE CASCADE).';
END $$;

COMMIT;
