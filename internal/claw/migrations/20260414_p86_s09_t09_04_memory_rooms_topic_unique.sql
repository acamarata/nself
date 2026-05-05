-- P86 S09 T09-04: Add topic_id column + UNIQUE (user_id, topic_id) on np_claw_memory_rooms.
-- Prevents race in autoAssignMemoryRooms where two concurrent batch cycles create
-- duplicate rooms for the same (user_id, topic_id).
--
-- Strategy:
--   1. Add topic_id column (nullable FK to np_claw_topics) if missing.
--   2. Backfill topic_id by matching room.name -> topic.name per user_id.
--   3. Dedup rooms: for each (user_id, topic_id) with topic_id NOT NULL,
--      keep the oldest row; repoint memories from dup rooms to the keeper
--      then delete dup rooms.
--   4. Add partial UNIQUE index on (user_id, topic_id) WHERE topic_id IS NOT NULL.
--      Partial so pre-existing rooms without a topic_id stay valid.

-- 1. Add column
ALTER TABLE np_claw_memory_rooms
    ADD COLUMN IF NOT EXISTS topic_id UUID REFERENCES np_claw_topics(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_memory_rooms_topic ON np_claw_memory_rooms(topic_id) WHERE topic_id IS NOT NULL;

-- 2. Backfill topic_id from topic name match (best-effort — only for rows missing topic_id).
UPDATE np_claw_memory_rooms r
SET topic_id = t.id
FROM np_claw_topics t
WHERE r.topic_id IS NULL
  AND r.user_id = t.user_id
  AND r.name = t.name;

-- 3. Dedup: repoint memories to the keeper (oldest) room for each (user_id, topic_id) pair,
-- then delete the non-keeper duplicates.
WITH ranked AS (
    SELECT id, user_id, topic_id,
           ROW_NUMBER() OVER (PARTITION BY user_id, topic_id ORDER BY created_at ASC, id ASC) AS rn
    FROM np_claw_memory_rooms
    WHERE topic_id IS NOT NULL
),
keepers AS (
    SELECT user_id, topic_id, id AS keeper_id FROM ranked WHERE rn = 1
),
dups AS (
    SELECT r.id AS dup_id, k.keeper_id
    FROM ranked r
    JOIN keepers k ON k.user_id = r.user_id AND k.topic_id = r.topic_id
    WHERE r.rn > 1
)
UPDATE np_claw_memories m
SET room_id = d.keeper_id
FROM dups d
WHERE m.room_id = d.dup_id;

WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY user_id, topic_id ORDER BY created_at ASC, id ASC) AS rn
    FROM np_claw_memory_rooms
    WHERE topic_id IS NOT NULL
)
DELETE FROM np_claw_memory_rooms
WHERE id IN (SELECT id FROM ranked WHERE rn > 1);

-- 4. Partial UNIQUE index (acts as unique constraint for ON CONFLICT targeting).
CREATE UNIQUE INDEX IF NOT EXISTS uq_memory_rooms_user_topic
    ON np_claw_memory_rooms (user_id, topic_id)
    WHERE topic_id IS NOT NULL;
