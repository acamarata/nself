-- P99 T02 (PLUG-02): Rollback dead letter queue.

DROP TRIGGER IF EXISTS np_claw_dead_letter_updated_at ON np_claw_dead_letter;
DROP FUNCTION IF EXISTS np_claw_dead_letter_set_updated_at();
DROP INDEX  IF EXISTS np_claw_dead_letter_user_pending_idx;
DROP INDEX  IF EXISTS np_claw_dead_letter_retry_idx;
DROP TABLE  IF EXISTS np_claw_dead_letter;
