-- Sprint 07 / N6: Skills marketplace.
-- Add is_public and published_at so users can optionally share their learned
-- skills with others through the marketplace listing.

ALTER TABLE np_claw_skills
    ADD COLUMN IF NOT EXISTS is_public     BOOLEAN       NOT NULL DEFAULT false;

ALTER TABLE np_claw_skills
    ADD COLUMN IF NOT EXISTS published_at  TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_np_claw_skills_public
    ON np_claw_skills(is_public, success_count DESC)
    WHERE is_public = true;
