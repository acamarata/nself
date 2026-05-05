-- DOWN
-- Rollback for 1742200000_pair_attempts.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- delete from np_claw.pair_attempts
where attempted_at < now()... cannot be auto-reversed; verify manually if rollback needed.
DROP INDEX IF EXISTS pair_attempts_ip_hash_idx;
DROP TABLE IF EXISTS np_claw.pair_attempts CASCADE;
