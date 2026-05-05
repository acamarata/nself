-- DOWN
-- Rollback for m_p88_c6_vocab.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_vocab_synonyms_gin;
DROP INDEX IF EXISTS idx_vocab_term_trgm;
DROP INDEX IF EXISTS idx_vocab_category;
ALTER TABLE IF EXISTS np_claw_user_vocabulary DROP COLUMN IF EXISTS synonyms;
ALTER TABLE IF EXISTS np_claw_user_vocabulary DROP COLUMN IF EXISTS source;
