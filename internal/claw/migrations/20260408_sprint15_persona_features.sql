-- Sprint 15: Agent & Persona Features tables

-- T15.5: Health Coach check-in data
CREATE TABLE IF NOT EXISTS np_claw_health_checkins (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    check_type  TEXT NOT NULL DEFAULT 'morning', -- morning, workout, supplement, custom
    data        JSONB NOT NULL DEFAULT '{}',
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_claw_health_checkins_user ON np_claw_health_checkins(user_id, created_at DESC);

-- T15.7: Learning Tutor spaced repetition cards
CREATE TABLE IF NOT EXISTS np_claw_tutor_cards (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       TEXT NOT NULL,
    question      TEXT NOT NULL,
    answer        TEXT NOT NULL,
    material_id   TEXT,
    ease_factor   DOUBLE PRECISION NOT NULL DEFAULT 2.5,
    interval_days INTEGER NOT NULL DEFAULT 1,
    repetitions   INTEGER NOT NULL DEFAULT 0,
    next_review   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_claw_tutor_cards_due ON np_claw_tutor_cards(user_id, next_review);

-- T15.9: Daily Mentor habits
CREATE TABLE IF NOT EXISTS np_claw_habits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    name            TEXT NOT NULL,
    frequency       TEXT NOT NULL DEFAULT 'daily', -- daily, weekly
    streak          INTEGER NOT NULL DEFAULT 0,
    best_streak     INTEGER NOT NULL DEFAULT 0,
    last_completed  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_claw_habits_user ON np_claw_habits(user_id);

CREATE TABLE IF NOT EXISTS np_claw_habit_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id     UUID NOT NULL REFERENCES np_claw_habits(id) ON DELETE CASCADE,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes        TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_claw_habit_logs_habit ON np_claw_habit_logs(habit_id, completed_at DESC);

-- T15.9: Daily Mentor reflections
CREATE TABLE IF NOT EXISTS np_claw_reflections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    wins        TEXT NOT NULL DEFAULT '',
    challenges  TEXT NOT NULL DEFAULT '',
    gratitude   TEXT NOT NULL DEFAULT '',
    sentiment   TEXT NOT NULL DEFAULT 'neutral',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_claw_reflections_user ON np_claw_reflections(user_id, created_at DESC);

-- T15.13: Job Hunter CV data
CREATE TABLE IF NOT EXISTS np_claw_cv_data (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL UNIQUE,
    raw_content     TEXT NOT NULL DEFAULT '',
    structured_data JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- T15.15: Marketplace ratings
CREATE TABLE IF NOT EXISTS np_claw_marketplace_ratings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id   UUID NOT NULL,
    user_id      TEXT NOT NULL,
    rating       INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(persona_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_claw_marketplace_ratings_persona ON np_claw_marketplace_ratings(persona_id);

-- T15.15: Marketplace installed personas tracking
CREATE TABLE IF NOT EXISTS np_claw_marketplace_installs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id   UUID NOT NULL,
    user_id      TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(persona_id, user_id)
);

-- T15.11: Domain Expert config
CREATE TABLE IF NOT EXISTS np_claw_domain_expert_config (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      TEXT NOT NULL UNIQUE,
    topic        TEXT NOT NULL DEFAULT '',
    materials    JSONB NOT NULL DEFAULT '[]',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add category column to personas if missing
DO $$ BEGIN
    ALTER TABLE np_claw_personas ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'General';
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Add rating_count to marketplace
DO $$ BEGIN
    ALTER TABLE np_claw_persona_marketplace ADD COLUMN IF NOT EXISTS rating_count INTEGER NOT NULL DEFAULT 0;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Add tokens_used and messages_today to agent dashboard view support
DO $$ BEGIN
    ALTER TABLE np_claw_agent_spend ADD COLUMN IF NOT EXISTS tokens_used BIGINT NOT NULL DEFAULT 0;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
