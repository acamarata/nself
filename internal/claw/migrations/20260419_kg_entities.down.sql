-- DOWN
-- Rollback for 20260419_kg_entities.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
DROP POLICY IF EXISTS kg_entities_user_isolation ON np_claw_kg_entities;
ALTER TABLE IF EXISTS np_claw_kg_entities DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_kg_entities_user_type;
DROP INDEX IF EXISTS idx_kg_entities_user_name;
DROP TABLE IF EXISTS np_claw_kg_entities CASCADE;
