-- DOWN
-- Rollback for 20260420_user_vocabulary.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
DROP POLICY IF EXISTS vocab_user_isolation ON np_claw_user_vocabulary;
ALTER TABLE IF EXISTS np_claw_user_vocabulary DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_vocab_user_conf;
DROP TABLE IF EXISTS np_claw_user_vocabulary CASCADE;
