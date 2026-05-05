-- P87 Sprint 01, Ticket 01.T09: Retrieval Profiles Table
-- Spec §4.9

-- UP

CREATE TABLE IF NOT EXISTS np_claw_retrieval_profiles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID,
    query_class     TEXT NOT NULL CHECK (query_class IN ('factoid','narrative','agent_tool','briefing','search')),
    w_vector        REAL NOT NULL DEFAULT 1.0,
    w_fts           REAL NOT NULL DEFAULT 0.8,
    w_trgm          REAL NOT NULL DEFAULT 0.3,
    w_ltree         REAL NOT NULL DEFAULT 0.5,
    w_recency       REAL NOT NULL DEFAULT 0.4,
    rrf_k           INT NOT NULL DEFAULT 60,
    CONSTRAINT uq_retrieval_user_class UNIQUE (user_id, query_class)
);

-- Seed 5 default profiles (user_id = NULL for global defaults)
INSERT INTO np_claw_retrieval_profiles (id, user_id, query_class, w_vector, w_fts, w_trgm, w_ltree, w_recency, rrf_k)
VALUES
    (gen_random_uuid(), NULL, 'factoid',    1.0, 0.9, 0.5, 0.3, 0.2, 60),
    (gen_random_uuid(), NULL, 'narrative',  1.0, 0.7, 0.2, 0.6, 0.6, 60),
    (gen_random_uuid(), NULL, 'agent_tool', 0.8, 1.0, 0.4, 0.3, 0.3, 60),
    (gen_random_uuid(), NULL, 'briefing',   0.9, 0.8, 0.3, 0.5, 0.8, 60),
    (gen_random_uuid(), NULL, 'search',     0.7, 1.0, 0.8, 0.4, 0.3, 60)
ON CONFLICT (user_id, query_class) DO NOTHING;

-- DOWN
-- DROP TABLE IF EXISTS np_claw_retrieval_profiles;
