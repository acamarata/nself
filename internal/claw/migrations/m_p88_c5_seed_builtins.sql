-- P88 Block C, Sprint 13, T13-01b: Seed built-in retrieval profiles
-- Spec §6.4 — 6 built-in profiles (read-only, is_builtin = true)

-- UP

-- Mark existing global defaults as builtin
UPDATE np_claw_retrieval_profiles
SET is_builtin = true, description = 'System default for ' || query_class
WHERE user_id IS NULL AND is_builtin = false;

-- Insert the 6 named built-in profiles (user_id = NULL, is_builtin = true)
INSERT INTO np_claw_retrieval_profiles
    (user_id, query_class, w_vector, w_fts, w_trgm, w_ltree, w_recency, rrf_k, is_builtin, description, score_floor, max_results)
VALUES
    (NULL, 'factoid',    1.0, 0.9, 0.5, 0.3, 0.2, 60, true, 'Default: balanced factoid retrieval', 0, 50),
    (NULL, 'narrative',  1.0, 0.7, 0.2, 0.6, 0.6, 60, true, 'Recent-bias: favors newer memories for narrative recall', 0, 50),
    (NULL, 'agent_tool', 0.0, 0.0, 0.0, 0.0, 0.0, 60, true, 'Semantic-only: pure vector similarity', 0, 50),
    (NULL, 'briefing',   0.0, 1.0, 0.0, 0.0, 0.0, 60, true, 'FTS-only: pure full-text search', 0, 50),
    (NULL, 'search',     0.7, 0.5, 0.3, 1.0, 0.2, 60, true, 'Ltree-strict: topic-path focused retrieval', 0, 50)
ON CONFLICT (user_id, query_class) DO UPDATE SET
    is_builtin = true,
    description = EXCLUDED.description;

-- Add a multi-channel built-in (new query_class)
-- First relax the CHECK constraint to allow 'multi_channel'
ALTER TABLE np_claw_retrieval_profiles DROP CONSTRAINT IF EXISTS np_claw_retrieval_profiles_query_class_check;
ALTER TABLE np_claw_retrieval_profiles ADD CONSTRAINT np_claw_retrieval_profiles_query_class_check
    CHECK (query_class IN ('factoid','narrative','agent_tool','briefing','search','multi_channel'));

INSERT INTO np_claw_retrieval_profiles
    (user_id, query_class, w_vector, w_fts, w_trgm, w_ltree, w_recency, rrf_k, is_builtin, description, score_floor, max_results)
VALUES
    (NULL, 'multi_channel', 0.8, 0.8, 0.5, 0.5, 0.5, 60, true, 'Multi-channel: equal weight across all signals', 0, 50)
ON CONFLICT (user_id, query_class) DO NOTHING;

-- DOWN
-- DELETE FROM np_claw_retrieval_profiles WHERE is_builtin = true;
