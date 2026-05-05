-- P97 W43 — DB-M3 down migration: drop FK and revert user_id to TEXT.

BEGIN;

ALTER TABLE np_claw_push_retry_queue
  DROP CONSTRAINT IF EXISTS np_claw_push_retry_queue_user_fkey;

-- Revert column type UUID → TEXT
DO $$
DECLARE
  v_col_type TEXT;
BEGIN
  SELECT data_type
    FROM information_schema.columns
   WHERE table_schema = 'public'
     AND table_name   = 'np_claw_push_retry_queue'
     AND column_name  = 'user_id'
    INTO v_col_type;

  IF v_col_type = 'uuid' THEN
    ALTER TABLE np_claw_push_retry_queue
      ALTER COLUMN user_id TYPE TEXT USING user_id::text;
  END IF;
END $$;

COMMIT;
