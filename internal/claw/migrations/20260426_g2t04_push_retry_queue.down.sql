-- G2-T04 rollback
DROP INDEX IF EXISTS idx_claw_push_retry_user;
DROP INDEX IF EXISTS idx_claw_push_retry_due;
DROP TABLE IF EXISTS np_claw_push_retry_queue;
