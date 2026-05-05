-- DOWN
-- Rollback for 090b_p88_skill_marketplace.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_skill_ratings_slug;
DROP TABLE IF EXISTS np_claw_skill_ratings CASCADE;
DROP INDEX IF EXISTS idx_skill_installs_slug;
DROP INDEX IF EXISTS idx_skill_installs_user;
DROP TABLE IF EXISTS np_claw_skill_installs CASCADE;
DROP INDEX IF EXISTS idx_skill_listings_slug;
DROP INDEX IF EXISTS idx_skill_listings_featured;
DROP INDEX IF EXISTS idx_skill_listings_category;
DROP TABLE IF EXISTS np_claw_skill_listings CASCADE;
