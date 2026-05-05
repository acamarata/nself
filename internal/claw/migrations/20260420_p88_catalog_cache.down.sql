-- DOWN
-- Rollback for 20260420_p88_catalog_cache.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_user_preferences DROP COLUMN IF EXISTS mock_mode;
-- (derived from DO block — review manually)
DROP POLICY IF EXISTS catalog_cache_user_isolation ON np_claw_tool_catalog_cache;
ALTER TABLE IF EXISTS np_claw_tool_catalog_cache DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS np_claw_tool_catalog_cache CASCADE;
