-- P86 T13: Add UNIQUE constraint on np_claw_memory_rooms(user_id, name)
-- Prevents duplicate rooms when concurrent batch cycles run simultaneously.

-- First remove any existing duplicates, keeping the oldest (smallest id by creation).
DELETE FROM np_claw_memory_rooms
WHERE id NOT IN (
    SELECT DISTINCT ON (user_id, name) id
    FROM np_claw_memory_rooms
    ORDER BY user_id, name, created_at ASC
);

ALTER TABLE np_claw_memory_rooms
    ADD CONSTRAINT uq_memory_rooms_user_name UNIQUE (user_id, name);
