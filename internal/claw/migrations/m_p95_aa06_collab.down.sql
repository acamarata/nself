-- Rollback P95 AA06 collab migration.
BEGIN;
DELETE FROM np_feature_flags WHERE flag = 'claw_collab';
DROP TABLE IF EXISTS np_claw_mentions;
DROP TABLE IF EXISTS np_claw_team_kb;
DROP TABLE IF EXISTS np_claw_shared_conversations;
COMMIT;
