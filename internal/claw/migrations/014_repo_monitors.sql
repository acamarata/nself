-- T-2176: Upstream repository monitor
-- Tracks GitHub repos for new releases, PRs, issues and scores relevance via AI.

CREATE TABLE IF NOT EXISTS np_claw_monitors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_url        TEXT NOT NULL,
    repo_owner      TEXT NOT NULL,
    repo_name       TEXT NOT NULL,
    check_interval_hours INTEGER NOT NULL DEFAULT 24,
    last_checked_at TIMESTAMPTZ,
    last_release    TEXT,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    filters         JSONB NOT NULL DEFAULT '{"releases": true, "issues": false, "pull_requests": false}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (repo_owner, repo_name)
);

CREATE TABLE IF NOT EXISTS np_claw_monitor_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id      UUID NOT NULL REFERENCES np_claw_monitors(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL, -- 'release', 'issue', 'pull_request'
    event_id        TEXT NOT NULL, -- GitHub event ID or tag name
    title           TEXT NOT NULL,
    url             TEXT NOT NULL,
    body_preview    TEXT,
    relevance_score REAL,          -- 0.0-1.0, AI-scored
    relevance_reason TEXT,
    notified        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (monitor_id, event_type, event_id)
);

CREATE INDEX IF NOT EXISTS idx_claw_monitors_enabled ON np_claw_monitors(enabled) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_claw_monitor_events_monitor ON np_claw_monitor_events(monitor_id);
CREATE INDEX IF NOT EXISTS idx_claw_monitor_events_relevance ON np_claw_monitor_events(relevance_score) WHERE relevance_score IS NOT NULL;
