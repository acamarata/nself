-- DOWN
-- Rollback for 20260414_p87_sprint37_security.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_auth_failed_logins_ip;
DROP TABLE IF EXISTS np_auth_failed_logins CASCADE;
DROP INDEX IF EXISTS idx_np_auth_refresh_user;
DROP INDEX IF EXISTS idx_np_auth_refresh_chain;
DROP INDEX IF EXISTS idx_np_auth_refresh_session;
DROP TABLE IF EXISTS np_auth_refresh_tokens CASCADE;
DROP TABLE IF EXISTS np_auth_devices CASCADE;
DROP TABLE IF EXISTS np_auth_mfa CASCADE;
DROP TABLE IF EXISTS np_sso_role_map CASCADE;
DROP TABLE IF EXISTS np_sso_config CASCADE;
DROP INDEX IF EXISTS idx_np_claw_user_keys_blind;
DROP TABLE IF EXISTS np_claw_user_keys CASCADE;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS blind_index;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS scheme;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS nonce;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS dek_wrapped;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS ciphertext;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS blind_index;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS scheme;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS nonce;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS dek_wrapped;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS ciphertext;
DROP INDEX IF EXISTS idx_np_secret_rotation_dual;
DROP INDEX IF EXISTS idx_np_secret_rotation_next;
DROP TABLE IF EXISTS np_secret_rotation CASCADE;
