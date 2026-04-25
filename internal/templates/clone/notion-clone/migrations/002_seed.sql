-- notion-clone demo data
INSERT INTO np_workspaces (id, owner_id, name)
VALUES ('00000000-2000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', 'My Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO np_pages (id, workspace_id, title, created_by)
VALUES ('00000000-2001-0000-0000-000000000001', '00000000-2000-0000-0000-000000000001', 'Getting Started', '00000000-0000-0000-0000-000000000010')
ON CONFLICT (id) DO NOTHING;

INSERT INTO np_blocks (page_id, type, content, position)
VALUES ('00000000-2001-0000-0000-000000000001', 'paragraph', '{"text": "Welcome to your workspace! Start editing this page."}', 0)
ON CONFLICT DO NOTHING;
