-- Down migration: drop media tables in reverse dependency order
DROP TABLE IF EXISTS np_claw_media_events;
DROP TABLE IF EXISTS np_claw_media_embeddings;
DROP TABLE IF EXISTS np_claw_attachments;
