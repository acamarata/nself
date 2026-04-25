-- discord-clone demo data
INSERT INTO np_servers (id, owner_id, name)
VALUES ('00000000-1000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', 'Demo Community')
ON CONFLICT (id) DO NOTHING;

INSERT INTO np_channels (id, server_id, name, type, position)
VALUES
    ('00000000-1001-0000-0000-000000000001', '00000000-1000-0000-0000-000000000001', 'general', 'text', 0),
    ('00000000-1001-0000-0000-000000000002', '00000000-1000-0000-0000-000000000001', 'announcements', 'announcement', 1)
ON CONFLICT (id) DO NOTHING;
