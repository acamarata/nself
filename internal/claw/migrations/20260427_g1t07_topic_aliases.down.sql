-- G1-T07 down: drop np_claw_topic_aliases.

DROP POLICY IF EXISTS topic_aliases_user_isolation ON np_claw_topic_aliases;
DROP INDEX IF EXISTS idx_np_claw_topic_aliases_embedding_hnsw;
DROP INDEX IF EXISTS idx_np_claw_topic_aliases_embedding_ivf;
DROP INDEX IF EXISTS idx_np_claw_topic_aliases_user_id;
DROP INDEX IF EXISTS idx_np_claw_topic_aliases_topic_id;
DROP TABLE IF EXISTS np_claw_topic_aliases;
