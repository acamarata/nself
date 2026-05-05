-- P88 Block C, Sprint 13, T13-06: C6 Vocabulary extensions
-- Spec §7.3

-- UP

-- Add new columns to existing vocab table
ALTER TABLE np_claw_user_vocabulary
    ADD COLUMN IF NOT EXISTS source     TEXT NOT NULL DEFAULT 'auto',
    ADD COLUMN IF NOT EXISTS synonyms   TEXT[] NOT NULL DEFAULT '{}';

-- Update last_used_at to have a default if missing rows exist
-- (already has DEFAULT now() from original migration)

-- Indexes for C6 queries
CREATE INDEX IF NOT EXISTS idx_vocab_category
    ON np_claw_user_vocabulary (user_id, category, usage_count DESC);

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_vocab_term_trgm
    ON np_claw_user_vocabulary USING GIN (term gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_vocab_synonyms_gin
    ON np_claw_user_vocabulary USING GIN (synonyms);

-- DOWN
-- ALTER TABLE np_claw_user_vocabulary DROP COLUMN IF EXISTS source, DROP COLUMN IF EXISTS synonyms;
-- DROP INDEX IF EXISTS idx_vocab_category;
-- DROP INDEX IF EXISTS idx_vocab_term_trgm;
-- DROP INDEX IF EXISTS idx_vocab_synonyms_gin;
