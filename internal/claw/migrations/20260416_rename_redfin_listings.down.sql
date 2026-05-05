-- DOWN
-- Rollback for 20260416_rename_redfin_listings.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_redfin_listings RENAME TO nclaw_redfin_listings;
