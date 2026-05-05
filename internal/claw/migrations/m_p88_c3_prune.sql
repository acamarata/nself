-- P88 Sprint 11: C3 Pruning production surface
-- T11-01: ALTER prune_candidates + CREATE prune_policy

-- New columns on existing table.
ALTER TABLE np_claw_prune_candidates
  ADD COLUMN IF NOT EXISTS defer_until   timestamptz,
  ADD COLUMN IF NOT EXISTS rejected_at   timestamptz,
  ADD COLUMN IF NOT EXISTS reason_note   text;

-- Composite index for filtered queue listing.
CREATE INDEX IF NOT EXISTS idx_prune_candidates_filter
  ON np_claw_prune_candidates (user_id, reason, score DESC)
  WHERE status = 'pending'
    AND (defer_until IS NULL OR defer_until < now());

-- Age-based index for sorting by creation time.
CREATE INDEX IF NOT EXISTS idx_prune_candidates_age
  ON np_claw_prune_candidates (user_id, created_at);

-- Per-user prune policy table.
CREATE TABLE IF NOT EXISTS np_claw_prune_policy (
  id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                 uuid NOT NULL UNIQUE,
  enabled                 boolean NOT NULL DEFAULT true,
  scan_frequency_hours    int NOT NULL DEFAULT 168,
  auto_threshold_score    double precision NOT NULL DEFAULT 0.8,
  auto_threshold_age_days int NOT NULL DEFAULT 365,
  reasons_enabled         text[] NOT NULL DEFAULT ARRAY['low_confidence','stale','contradicted','superseded','never_retrieved'],
  exempt_topic_paths      text[] NOT NULL DEFAULT '{}',
  exempt_profile_ids      text[] NOT NULL DEFAULT '{}',
  notify_on_candidates    boolean NOT NULL DEFAULT true,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_prune_policy_user ON np_claw_prune_policy (user_id);
