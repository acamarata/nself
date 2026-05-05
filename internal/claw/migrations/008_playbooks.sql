-- T-1073: np_claw_playbooks table for incident response playbooks

CREATE TABLE IF NOT EXISTS np_claw_playbooks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  description TEXT,
  trigger_keywords TEXT[],
  steps JSONB NOT NULL DEFAULT '[]',
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_np_claw_playbooks_name ON np_claw_playbooks(name);

-- Seed: basic incident response playbook
INSERT INTO np_claw_playbooks(name, description, trigger_keywords, steps) VALUES (
  'incident_response',
  'Basic incident response playbook for service outages',
  ARRAY['incident', 'outage', 'down', 'critical'],
  '[
    {"step": 1, "action": "notify_team", "message": "Incident detected: {description}"},
    {"step": 2, "action": "check_services", "command": "nself status"},
    {"step": 3, "action": "collect_logs", "tail_lines": 100},
    {"step": 4, "action": "ai_diagnose", "prompt": "Diagnose this incident: {logs}"},
    {"step": 5, "action": "report_summary", "send_to": "telegram"}
  ]'::jsonb
) ON CONFLICT DO NOTHING;

INSERT INTO np_claw_playbooks(name, description, trigger_keywords, steps) VALUES (
  'daily_digest',
  'Generate daily digest of server activity',
  ARRAY['digest', 'daily', 'summary'],
  '[
    {"step": 1, "action": "collect_metrics", "period": "24h"},
    {"step": 2, "action": "ai_summarize", "prompt": "Summarize the last 24h of activity"},
    {"step": 3, "action": "send_digest", "send_to": "telegram"}
  ]'::jsonb
) ON CONFLICT DO NOTHING;
