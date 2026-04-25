-- substack-clone demo data
INSERT INTO np_newsletters (id, owner_id, name, slug, description)
VALUES ('00000000-4000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', 'My Newsletter', 'my-newsletter', 'A sample newsletter.')
ON CONFLICT (id) DO NOTHING;

INSERT INTO np_tiers (id, newsletter_id, name, is_free)
VALUES ('00000000-4001-0000-0000-000000000001', '00000000-4000-0000-0000-000000000001', 'Free', TRUE)
ON CONFLICT (id) DO NOTHING;
