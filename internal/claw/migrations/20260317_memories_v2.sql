-- Migration: memories v2 — typed, classified, vectorized knowledge base
-- Requires pgvector extension

-- Add pgvector if not present
CREATE EXTENSION IF NOT EXISTS vector;

-- Add new columns to np_claw_memories
ALTER TABLE np_claw_memories
  ADD COLUMN IF NOT EXISTS memory_type TEXT CHECK (memory_type IN ('fact','preference','decision','pattern','relationship')) DEFAULT 'fact',
  ADD COLUMN IF NOT EXISTS subject TEXT,
  ADD COLUMN IF NOT EXISTS source_type TEXT CHECK (source_type IN ('explicit','inferred','pattern')) DEFAULT 'explicit',
  ADD COLUMN IF NOT EXISTS source_conversation_id UUID,
  ADD COLUMN IF NOT EXISTS superseded_by UUID REFERENCES np_claw_memories(id),
  ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS use_count INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS content_embedding VECTOR(1536),
  ADD COLUMN IF NOT EXISTS blind_index TEXT;

-- HNSW index for semantic search
CREATE INDEX IF NOT EXISTS memories_embedding_hnsw ON np_claw_memories
  USING hnsw (content_embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);

-- Index for type-faceted queries
CREATE INDEX IF NOT EXISTS memories_type_idx ON np_claw_memories (memory_type) WHERE superseded_by IS NULL;
CREATE INDEX IF NOT EXISTS memories_subject_idx ON np_claw_memories (entity_id, subject) WHERE superseded_by IS NULL;

-- Migrate existing rows: set defaults for new non-null fields
UPDATE np_claw_memories
  SET memory_type = 'fact',
      source_type = 'explicit',
      confidence = COALESCE(confidence, 1.0)
  WHERE memory_type IS NULL;

-- Make memory_type NOT NULL now that rows are backfilled
ALTER TABLE np_claw_memories ALTER COLUMN memory_type SET NOT NULL;
ALTER TABLE np_claw_memories ALTER COLUMN source_type SET NOT NULL;
