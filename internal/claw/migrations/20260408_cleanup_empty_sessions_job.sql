-- Seed proactive job: cleanup empty sessions every 6 hours.
-- Deletes sessions with zero messages older than 24 hours.

INSERT INTO np_claw_proactive_jobs (
  id, job_type, enabled, cron_expression, config, quiet_hours_start, quiet_hours_end
) VALUES (
  gen_random_uuid(),
  'cleanup_empty_sessions',
  true,
  '0 */6 * * *',
  '{}',
  0, 0
) ON CONFLICT (job_type) DO NOTHING;
