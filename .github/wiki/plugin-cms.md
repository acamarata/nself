# CMS Plugin

> Headless CMS with content types, posts, categories, versioning, and GraphQL API. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install cms
```

## What It Does

Adds a full headless CMS to your nSelf backend. Define custom content types (articles, products, events, etc.), manage posts with rich text, organize content with categories and tags, and publish through a versioned workflow. All content is accessible via Hasura GraphQL for any frontend. Supports media management via MinIO.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CMS_PORT` | `3501` | CMS service port |
| `CMS_MEDIA_ENABLED` | `true` | Enable media library (requires MinIO) |
| `CMS_DRAFT_ENABLED` | `true` | Enable draft/publish workflow |
| `CMS_VERSION_HISTORY` | `10` | Versions to retain per content item |
| `CMS_DEFAULT_LOCALE` | `en` | Default content locale |

## Ports

| Port | Purpose |
|------|---------|
| 3501 | CMS REST API and admin UI |

## Database Tables

8 tables added to your Postgres database:
- `np_cms_content_types` — content type definitions
- `np_cms_posts` — content items
- `np_cms_post_versions` — version history
- `np_cms_categories` — category tree
- `np_cms_tags` — tag definitions
- `np_cms_post_tags` — post-tag relationships
- `np_cms_media` — media library records
- `np_cms_locales` — locale configurations

## Nginx Routes

| Route | Target |
|-------|--------|
| `/cms/` | CMS management API |
| `/cms/admin` | CMS admin UI |
