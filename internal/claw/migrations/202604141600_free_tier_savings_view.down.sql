-- DOWN
-- Rollback for 202604141600_free_tier_savings_view.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP TABLE IF EXISTS np_ai_free_tier_quota CASCADE;
DROP INDEX IF EXISTS idx_recommendation_actioned_user;
DROP TABLE IF EXISTS np_ai_recommendation_actioned CASCADE;
DROP INDEX IF EXISTS idx_cost_override_audit_created;
DROP TABLE IF EXISTS np_ai_cost_override_audit CASCADE;
DROP VIEW IF EXISTS np_ai_free_tier_savings_daily CASCADE;
