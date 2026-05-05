-- DOWN
-- Rollback for 003_api_gateway.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_system_prompts_user_id;
DROP TABLE IF EXISTS np_claw_system_prompts CASCADE;
DROP INDEX IF EXISTS idx_np_claw_api_usage_created_at;
DROP INDEX IF EXISTS idx_np_claw_api_usage_key_id;
DROP TABLE IF EXISTS np_claw_api_usage CASCADE;
DROP INDEX IF EXISTS idx_np_claw_api_keys_hash;
DROP INDEX IF EXISTS idx_np_claw_api_keys_user_id;
DROP TABLE IF EXISTS np_claw_api_keys CASCADE;
