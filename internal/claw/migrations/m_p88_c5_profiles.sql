-- P88 Block C, Sprint 13, T13-01a: C5 Retrieval Profile extensions
-- Spec §6.4

-- UP

-- Add new columns to existing retrieval_profiles table
ALTER TABLE np_claw_retrieval_profiles
    ADD COLUMN IF NOT EXISTS is_builtin    BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS score_floor   NUMERIC NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_results   INT NOT NULL DEFAULT 50,
    ADD COLUMN IF NOT EXISTS description   TEXT,
    ADD COLUMN IF NOT EXISTS deleted_at    TIMESTAMPTZ;

-- A/B testing table
CREATE TABLE IF NOT EXISTS np_claw_profile_ab_tests (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID,
    query_class    TEXT NOT NULL,
    profile_a_id   UUID NOT NULL REFERENCES np_claw_retrieval_profiles(id),
    profile_b_id   UUID NOT NULL REFERENCES np_claw_retrieval_profiles(id),
    traffic_split  INT NOT NULL DEFAULT 50 CHECK (traffic_split >= 10 AND traffic_split <= 90),
    starts_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    ends_at        TIMESTAMPTZ NOT NULL,
    status         TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','completed','cancelled')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_profile_ab_active
    ON np_claw_profile_ab_tests (query_class, status) WHERE status = 'active';

-- Quality scoring daily rollup
CREATE TABLE IF NOT EXISTS np_claw_profile_quality_daily (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL,
    profile_id       UUID NOT NULL REFERENCES np_claw_retrieval_profiles(id),
    query_class      TEXT NOT NULL,
    day              DATE NOT NULL,
    queries          INT NOT NULL DEFAULT 0,
    avg_click_rank   NUMERIC DEFAULT 0,
    avg_dwell_seconds NUMERIC DEFAULT 0,
    thumbs_up        INT NOT NULL DEFAULT 0,
    thumbs_down      INT NOT NULL DEFAULT 0,
    quality_score    NUMERIC NOT NULL DEFAULT 0,
    CONSTRAINT uq_quality_daily UNIQUE (user_id, profile_id, query_class, day)
);

CREATE INDEX IF NOT EXISTS idx_quality_daily_profile
    ON np_claw_profile_quality_daily (profile_id, day DESC);

-- Soft delete index for profiles
CREATE INDEX IF NOT EXISTS idx_profiles_active
    ON np_claw_retrieval_profiles (user_id, query_class) WHERE deleted_at IS NULL;

-- DOWN
-- ALTER TABLE np_claw_retrieval_profiles DROP COLUMN IF EXISTS is_builtin, DROP COLUMN IF EXISTS score_floor, DROP COLUMN IF EXISTS max_results, DROP COLUMN IF EXISTS description, DROP COLUMN IF EXISTS deleted_at;
-- DROP TABLE IF EXISTS np_claw_profile_quality_daily;
-- DROP TABLE IF EXISTS np_claw_profile_ab_tests;
