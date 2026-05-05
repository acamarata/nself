-- Fix is_global default: memories without a topic should be globally visible.
-- The column was added with DEFAULT false, which made topic-less memories
-- invisible to both topic-scoped and global recall paths.

-- Change column default to true so all future inserts that omit the column
-- are automatically globally visible.
ALTER TABLE np_claw_memories
    ALTER COLUMN is_global SET DEFAULT true;

-- Backfill existing memories that have no topic assignment and are not
-- yet marked global. These were inserted with the wrong default.
UPDATE np_claw_memories
SET is_global = true
WHERE topic_id IS NULL
  AND is_global = false;
