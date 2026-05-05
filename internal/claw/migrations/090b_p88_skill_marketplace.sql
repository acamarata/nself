-- P88 Sprint 17 — Skill Marketplace tables
-- S17-T01: np_claw_skill_listings + np_claw_skill_installs

CREATE TABLE IF NOT EXISTS np_claw_skill_listings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,
    title           TEXT NOT NULL,
    author          TEXT NOT NULL DEFAULT '',
    author_verified BOOLEAN NOT NULL DEFAULT false,
    category        TEXT NOT NULL DEFAULT 'general',
    version         TEXT NOT NULL DEFAULT '0.1.0',
    tools_used      TEXT[] NOT NULL DEFAULT '{}',
    min_tier        TEXT NOT NULL DEFAULT 'free',
    description_md  TEXT NOT NULL DEFAULT '',
    readme_url      TEXT NOT NULL DEFAULT '',
    icon_url        TEXT NOT NULL DEFAULT '',
    install_count   INTEGER NOT NULL DEFAULT 0,
    rating_avg      NUMERIC(3,2) NOT NULL DEFAULT 0,
    rating_n        INTEGER NOT NULL DEFAULT 0,
    featured        BOOLEAN NOT NULL DEFAULT false,
    featured_at     TIMESTAMPTZ,
    published_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    repo_url        TEXT NOT NULL DEFAULT '',
    license         TEXT NOT NULL DEFAULT 'MIT'
);

CREATE INDEX IF NOT EXISTS idx_skill_listings_category ON np_claw_skill_listings(category);
CREATE INDEX IF NOT EXISTS idx_skill_listings_featured ON np_claw_skill_listings(featured) WHERE featured = true;
CREATE INDEX IF NOT EXISTS idx_skill_listings_slug ON np_claw_skill_listings(slug);

CREATE TABLE IF NOT EXISTS np_claw_skill_installs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      TEXT NOT NULL,
    slug         TEXT NOT NULL REFERENCES np_claw_skill_listings(slug),
    version      TEXT NOT NULL DEFAULT '0.1.0',
    enabled      BOOLEAN NOT NULL DEFAULT true,
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_skill_installs_user ON np_claw_skill_installs(user_id);
CREATE INDEX IF NOT EXISTS idx_skill_installs_slug ON np_claw_skill_installs(slug);

CREATE TABLE IF NOT EXISTS np_claw_skill_ratings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT NOT NULL,
    slug       TEXT NOT NULL REFERENCES np_claw_skill_listings(slug),
    rating     INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_skill_ratings_slug ON np_claw_skill_ratings(slug);
