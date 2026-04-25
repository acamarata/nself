-- zoom-clone demo data
INSERT INTO np_meetings (id, host_id, title, description)
VALUES ('00000000-5000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', 'Team Standup', 'Daily team standup meeting.')
ON CONFLICT (id) DO NOTHING;
