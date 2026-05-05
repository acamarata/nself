-- Migration: Claw BIOS tables (SP-20 C17a)
-- Creates np_claw_bios_config, np_claw_bios_snapshots, np_claw_tool_registry
-- Per spec C17 §4 Data Model.

-- BIOS boot-time configuration record (one per deploy + tenant).
CREATE TABLE IF NOT EXISTS np_claw_bios_config (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  provider_pool_hash     TEXT NOT NULL,
  schema_migration_hash  TEXT NOT NULL,
  pgvector_dim           INT  NOT NULL,
  hasura_endpoint        TEXT NOT NULL,
  plugin_registry_hash   TEXT NOT NULL,
  anti_drift_rules_hash  TEXT NOT NULL,
  captured_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  tenant_id              UUID REFERENCES np_tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bios_config_tenant_captured
  ON np_claw_bios_config (tenant_id, captured_at DESC);

-- BIOS periodic state snapshots — diff basis for drift detection.
CREATE TABLE IF NOT EXISTS np_claw_bios_snapshots (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  bios_config_id  UUID NOT NULL REFERENCES np_claw_bios_config(id) ON DELETE CASCADE,
  snapshot_data   JSONB NOT NULL,
  drift_detected  BOOLEAN NOT NULL DEFAULT FALSE,
  drift_fields    TEXT[],
  captured_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bios_snapshots_drift
  ON np_claw_bios_snapshots (drift_detected, captured_at DESC);
CREATE INDEX IF NOT EXISTS idx_bios_snapshots_config
  ON np_claw_bios_snapshots (bios_config_id, captured_at DESC);

-- Tool registry — each tool call is validated against this table by Layer 2.
CREATE TABLE IF NOT EXISTS np_claw_tool_registry (
  tool_name      TEXT PRIMARY KEY,
  plugin_owner   TEXT NOT NULL,
  json_schema    JSONB NOT NULL DEFAULT '{}',
  allowed_roles  TEXT[],
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deprecated_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tool_registry_active
  ON np_claw_tool_registry (tool_name)
  WHERE deprecated_at IS NULL;

-- Hasura Row-Level Security notes (enforced via Hasura metadata, not SQL):
--   np_claw_bios_config:     filter {"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}
--   np_claw_bios_snapshots:  join via bios_config_id (inherits tenant filter)
--   np_claw_tool_registry:   global (no tenant filter — registry is per-deploy)

-- Seed built-in tool registry entries for the standard claw tool set.
-- These mirror the built-in tools registered in internal/tools at startup.
INSERT INTO np_claw_tool_registry (tool_name, plugin_owner, json_schema, allowed_roles)
VALUES
  ('research', 'claw', '{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}', ARRAY['user','admin']),
  ('memory_search', 'claw', '{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer"}},"required":["query"]}', ARRAY['user','admin']),
  ('memory_store', 'claw', '{"type":"object","properties":{"content":{"type":"string"},"topic":{"type":"string"}},"required":["content"]}', ARRAY['user','admin']),
  ('calendar_read', 'claw', '{"type":"object","properties":{"range_days":{"type":"integer"}}}', ARRAY['user','admin']),
  ('email_read', 'claw', '{"type":"object","properties":{"limit":{"type":"integer"},"query":{"type":"string"}}}', ARRAY['user','admin']),
  ('python_exec', 'claw', '{"type":"object","properties":{"code":{"type":"string"}},"required":["code"]}', ARRAY['user','admin']),
  ('web_search', 'claw', '{"type":"object","properties":{"query":{"type":"string"},"num_results":{"type":"integer"}},"required":["query"]}', ARRAY['user','admin']),
  ('file_read', 'claw', '{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}', ARRAY['user','admin']),
  ('news_fetch', 'claw-news', '{"type":"object","properties":{"topic":{"type":"string"},"count":{"type":"integer"}}}', ARRAY['user','admin']),
  ('budget_query', 'claw-budget', '{"type":"object","properties":{"period":{"type":"string"},"category":{"type":"string"}}}', ARRAY['user','admin'])
ON CONFLICT (tool_name) DO NOTHING;
