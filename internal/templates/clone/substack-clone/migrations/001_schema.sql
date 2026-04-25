-- substack-clone schema
-- Tables: np_newsletters, np_posts, np_subscribers, np_tiers

CREATE TABLE IF NOT EXISTS np_newsletters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    slug TEXT UNIQUE NOT NULL,
    logo_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    newsletter_id UUID NOT NULL REFERENCES np_newsletters(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    price_monthly NUMERIC(10,2),
    price_annual NUMERIC(10,2),
    description TEXT,
    is_free BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS np_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    newsletter_id UUID NOT NULL REFERENCES np_newsletters(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    subtitle TEXT,
    content JSONB NOT NULL DEFAULT '{}',
    is_paywalled BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','archived')),
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_subscribers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    newsletter_id UUID NOT NULL REFERENCES np_newsletters(id) ON DELETE CASCADE,
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    email TEXT NOT NULL,
    tier_id UUID REFERENCES np_tiers(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','unsubscribed','bounced')),
    subscribed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(newsletter_id, email)
);

-- RLS
ALTER TABLE np_newsletters ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_subscribers ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_tiers ENABLE ROW LEVEL SECURITY;

CREATE POLICY np_newsletters_owner ON np_newsletters USING (owner_id = auth.uid());
CREATE POLICY np_newsletters_public ON np_newsletters FOR SELECT USING (TRUE);
CREATE POLICY np_posts_public ON np_posts FOR SELECT USING (status = 'published' AND is_paywalled = FALSE);
CREATE POLICY np_posts_author ON np_posts USING (author_id = auth.uid());
CREATE POLICY np_subscribers_owner ON np_subscribers USING (
    newsletter_id IN (SELECT id FROM np_newsletters WHERE owner_id = auth.uid())
);
CREATE POLICY np_tiers_public ON np_tiers FOR SELECT USING (TRUE);
CREATE POLICY np_tiers_owner ON np_tiers USING (
    newsletter_id IN (SELECT id FROM np_newsletters WHERE owner_id = auth.uid())
);

-- Indexes
CREATE INDEX IF NOT EXISTS np_posts_newsletter_idx ON np_posts(newsletter_id, published_at DESC);
CREATE INDEX IF NOT EXISTS np_subscribers_newsletter_idx ON np_subscribers(newsletter_id);
