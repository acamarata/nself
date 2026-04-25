-- notion-clone schema
-- Tables: np_workspaces, np_pages, np_blocks, np_page_members

CREATE TABLE IF NOT EXISTS np_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    icon TEXT DEFAULT '📝',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES np_workspaces(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES np_pages(id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT 'Untitled',
    icon TEXT DEFAULT '📄',
    cover_url TEXT,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    position INTEGER NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id UUID NOT NULL REFERENCES np_pages(id) ON DELETE CASCADE,
    type TEXT NOT NULL DEFAULT 'paragraph',
    content JSONB NOT NULL DEFAULT '{}',
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_page_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id UUID NOT NULL REFERENCES np_pages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('viewer','commenter','editor')),
    UNIQUE(page_id, user_id)
);

-- RLS
ALTER TABLE np_workspaces ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_pages ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_blocks ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_page_members ENABLE ROW LEVEL SECURITY;

CREATE POLICY np_workspaces_owner ON np_workspaces USING (owner_id = auth.uid());
CREATE POLICY np_pages_workspace ON np_pages FOR SELECT USING (
    workspace_id IN (SELECT id FROM np_workspaces WHERE owner_id = auth.uid())
    OR id IN (SELECT page_id FROM np_page_members WHERE user_id = auth.uid())
);
CREATE POLICY np_pages_owner ON np_pages USING (created_by = auth.uid());
CREATE POLICY np_blocks_page ON np_blocks FOR SELECT USING (
    page_id IN (SELECT id FROM np_pages WHERE created_by = auth.uid()
                UNION SELECT page_id FROM np_page_members WHERE user_id = auth.uid())
);
CREATE POLICY np_blocks_write ON np_blocks USING (
    page_id IN (SELECT id FROM np_pages WHERE created_by = auth.uid()
                UNION SELECT page_id FROM np_page_members WHERE user_id = auth.uid() AND role = 'editor')
);

-- Indexes
CREATE INDEX IF NOT EXISTS np_pages_workspace_idx ON np_pages(workspace_id);
CREATE INDEX IF NOT EXISTS np_pages_parent_idx ON np_pages(parent_id);
CREATE INDEX IF NOT EXISTS np_blocks_page_idx ON np_blocks(page_id, position);
