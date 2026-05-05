-- Phase 92: Enforce np_claw_* table naming conformance.
-- Renames any legacy nclaw_redfin_listings table (created dynamically by
-- mux structured_extract action) to the conforming np_claw_redfin_listings
-- name. Safe to apply multiple times.

ALTER TABLE IF EXISTS nclaw_redfin_listings RENAME TO np_claw_redfin_listings;
