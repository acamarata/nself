-- T-1428: Pair attempt audit table
-- Tracks all pairing code attempts for rate limiting and security audit.
-- IP addresses are hashed (SHA-256) for privacy. Code tokens are hashed for audit.

CREATE TABLE IF NOT EXISTS np_claw.pair_attempts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_hash      TEXT        NOT NULL,
    code_hash    TEXT        NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    success      BOOLEAN     NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS pair_attempts_ip_hash_idx
    ON np_claw.pair_attempts(ip_hash, attempted_at);

-- Auto-cleanup: delete entries older than 7 days (also runs at startup via server.rs).
DELETE FROM np_claw.pair_attempts
WHERE attempted_at < NOW() - INTERVAL '7 days';
