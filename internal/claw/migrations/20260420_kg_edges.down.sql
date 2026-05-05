-- DOWN
-- Rollback for 20260420_kg_edges.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- update np_claw_kg_predicates set inverse_of = 'parent_of' wh... cannot be auto-reversed; verify manually if rollback needed.
-- update np_claw_kg_predicates set inverse_of = 'child_of'  wh... cannot be auto-reversed; verify manually if rollback needed.
-- update np_claw_kg_predicates set inverse_of = 'before'    wh... cannot be auto-reversed; verify manually if rollback needed.
-- update np_claw_kg_predicates set inverse_of = 'after'     wh... cannot be auto-reversed; verify manually if rollback needed.
-- (derived from DO block — review manually)
DROP POLICY IF EXISTS kg_predicates_read_all ON np_claw_kg_predicates;
ALTER TABLE IF EXISTS np_claw_kg_predicates DISABLE ROW LEVEL SECURITY;
-- (derived from DO block — review manually)
DROP POLICY IF EXISTS kg_edges_user_isolation ON np_claw_kg_edges;
ALTER TABLE IF EXISTS np_claw_kg_edges DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_kg_edges_valid;
DROP INDEX IF EXISTS idx_kg_edges_predicate;
DROP INDEX IF EXISTS idx_kg_edges_object;
DROP INDEX IF EXISTS idx_kg_edges_subject;
DROP TABLE IF EXISTS np_claw_kg_edges CASCADE;
DROP TABLE IF EXISTS np_claw_kg_predicates CASCADE;
