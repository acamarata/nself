-- Sprint 07: Memory Cross-Channel
-- T-7704: source_channel column — tracks which channel (chat/email/calendar/notification/system)
--         produced each memory, enabling per-channel filtering and analytics.
-- T-7705: knowledge_distillation proactive job seed.

-- T-7704: Add source_channel to np_claw_memories
ALTER TABLE np_claw_memories
  ADD COLUMN IF NOT EXISTS source_channel TEXT NOT NULL DEFAULT 'chat';

CREATE INDEX IF NOT EXISTS idx_np_claw_memories_source_channel
  ON np_claw_memories (source_channel);

CREATE INDEX IF NOT EXISTS idx_np_claw_memories_entity_channel
  ON np_claw_memories (entity_id, source_channel)
  WHERE superseded_by IS NULL;

-- T-7705: seed weekly knowledge distillation job
INSERT INTO np_claw_proactive_jobs (job_type, cron_expression, next_run_at, config)
VALUES (
  'knowledge_distillation',
  '0 4 * * 0',
  NOW() + INTERVAL '1 day',
  '{"max_sessions": 50, "min_message_length": 20}'
)
ON CONFLICT (job_type) DO NOTHING;
