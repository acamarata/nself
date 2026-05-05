-- Used by startup migrator on fresh installs — full schema
-- (on existing installs, 20260317_memories_v2.sql runs ALTER)
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS np_claw_memories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_id TEXT NOT NULL,
  entity_type TEXT NOT NULL DEFAULT 'user',
  content TEXT NOT NULL,
  memory_type TEXT NOT NULL CHECK (memory_type IN ('fact','preference','decision','pattern','relationship')) DEFAULT 'fact',
  subject TEXT,
  context_hash TEXT,
  source TEXT NOT NULL DEFAULT 'extracted',
  source_type TEXT NOT NULL CHECK (source_type IN ('explicit','inferred','pattern')) DEFAULT 'explicit',
  source_conversation_id UUID,
  confidence REAL NOT NULL DEFAULT 1.0,
  times_reinforced INTEGER NOT NULL DEFAULT 1,
  superseded_by UUID REFERENCES np_claw_memories(id),
  expires_at TIMESTAMPTZ,
  use_count INT NOT NULL DEFAULT 0,
  last_used_at TIMESTAMPTZ,
  content_embedding VECTOR(1536),
  blind_index TEXT,
  metadata JSONB NOT NULL DEFAULT '{}',
  project_id UUID,
  source_channel TEXT NOT NULL DEFAULT 'chat',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_memories_entity_id ON np_claw_memories(entity_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_memories_project_id ON np_claw_memories(project_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_memories_entity_type ON np_claw_memories(entity_type);
CREATE INDEX IF NOT EXISTS idx_np_claw_memories_source_channel ON np_claw_memories(source_channel);
CREATE INDEX IF NOT EXISTS idx_np_claw_memories_entity_channel ON np_claw_memories(entity_id, source_channel) WHERE superseded_by IS NULL;
CREATE INDEX IF NOT EXISTS idx_np_claw_memories_confidence ON np_claw_memories(confidence DESC);
CREATE INDEX IF NOT EXISTS memories_embedding_hnsw ON np_claw_memories
  USING hnsw (content_embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);
CREATE INDEX IF NOT EXISTS memories_type_idx ON np_claw_memories (memory_type) WHERE superseded_by IS NULL;
CREATE INDEX IF NOT EXISTS memories_subject_idx ON np_claw_memories (entity_id, subject) WHERE superseded_by IS NULL;
