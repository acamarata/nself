-- S46-T06 schema drift reconciliation (claw)
-- Closes 93-DATA-51 item (3) np_claw_mux_accounts — queried by claw/internal/server/server.go
-- but only np_mux_accounts exists in the mux plugin.
-- Decision: alias via VIEW. Keeps mux as the canonical source, exposes a claw-namespaced read view.
-- Idempotent: CREATE OR REPLACE VIEW.

-- Guard: only create the view if the underlying mux table exists
-- (plugin may not be installed; view creation would fail otherwise).
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
     WHERE table_name = 'np_mux_accounts'
  ) THEN
    EXECUTE 'CREATE OR REPLACE VIEW np_claw_mux_accounts AS SELECT * FROM np_mux_accounts';
  END IF;
END $$;
