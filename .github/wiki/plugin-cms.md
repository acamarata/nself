# CMS Plugin

> Headless CMS with content types, posts, categories, versioning, and GraphQL API. **Pro plugin.**

> **Requires:** Pro license tier. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install cms
```

## What It Does

Adds a full headless CMS to your ɳSelf backend. Define custom content types (articles, products, events, etc.), manage posts with rich text, organize content with categories and tags, and publish through a versioned workflow. All content is accessible via Hasura GraphQL for any frontend.

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `DATABASE_URL` | yes | none | Postgres connection string |
| `CMS_PLUGIN_PORT` | no | `3501` | CMS service port |

## Ports

| Port | Purpose |
|------|---------|
| 3501 | CMS REST API |

## Database Tables

8 tables added to your Postgres database:
- `np_cms_content_types`, content type definitions
- `np_cms_posts`, content items
- `np_cms_post_versions`, version history
- `np_cms_categories`, category tree
- `np_cms_tags`, tag definitions
- `np_cms_post_categories`, post-category relationships
- `np_cms_post_tags`, post-tag relationships
- `np_cms_webhook_events`, webhook event log

All tables use `source_account_id` for multi-app isolation. Content is automatically
scoped to the calling app and is not visible across apps on the same backend.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/cms/` | CMS REST API |
