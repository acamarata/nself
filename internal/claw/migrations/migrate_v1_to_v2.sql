-- nself-claw v1 → v2 migration
-- Copies np_claw_sessions → cl_threads
-- Copies np_claw_messages → cl_messages
-- Copies np_claw_memory → cl_memory_blocks
-- ON CONFLICT DO NOTHING (additive, v1 tables preserved)
-- WARNING: embeddings NOT migrated — must regenerate after migration

-- Migrate sessions → threads (if np_claw_sessions exists)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'np_claw_sessions') THEN
    INSERT INTO cl_threads (id, user_id, namespace, persona_name, title, is_archived, created_at, updated_at)
    SELECT
      id,
      COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid),
      COALESCE(namespace, 'default'),
      COALESCE(persona_name, 'CamClaw'),
      COALESCE(title, 'Migrated Session'),
      COALESCE(is_archived, false),
      created_at,
      COALESCE(updated_at, created_at)
    FROM np_claw_sessions
    ON CONFLICT (id) DO NOTHING;
    RAISE NOTICE 'Migrated % rows from np_claw_sessions to cl_threads', (SELECT COUNT(*) FROM cl_threads);
  END IF;
END $$;

-- Migrate messages
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'np_claw_messages') THEN
    INSERT INTO cl_messages (id, thread_id, role, content, token_count, is_compressed, created_at)
    SELECT
      id,
      session_id,  -- np_claw_messages.session_id → cl_messages.thread_id
      role,
      content,
      COALESCE(token_count, 0),
      false,
      created_at
    FROM np_claw_messages
    WHERE session_id IN (SELECT id FROM cl_threads)
    ON CONFLICT (id) DO NOTHING;
    RAISE NOTICE 'Migrated % rows from np_claw_messages to cl_messages', (SELECT COUNT(*) FROM cl_messages);
  END IF;
END $$;

-- Migrate memory
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'np_claw_memory') THEN
    INSERT INTO cl_memory_blocks (id, thread_id, content, source_msg_ids, token_count, created_at)
    SELECT
      id,
      COALESCE(session_id, (SELECT id FROM cl_threads LIMIT 1)),
      content,
      '{}',  -- source_msg_ids unknown for v1 data
      COALESCE(token_count, length(content) / 4),
      created_at
    FROM np_claw_memory
    WHERE session_id IN (SELECT id FROM cl_threads)
    ON CONFLICT (id) DO NOTHING;
    RAISE NOTICE 'Migrated % rows from np_claw_memory to cl_memory_blocks';
  END IF;
END $$;

-- Note: embeddings in cl_messages.embedding and cl_memory_blocks.embedding
-- are NULL for migrated data. Run re-embedding job after migration.
