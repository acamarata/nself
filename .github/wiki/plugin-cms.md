# CMS Plugin

> Headless CMS with content types, posts, categories, versioning, and GraphQL API. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install cms
```

## What It Does

Adds a full headless CMS to your ɳSelf backend. Define custom content types (articles, products, events, etc.), manage posts with rich text, organize content with categories and tags, and publish through a versioned workflow. All content is accessible via Hasura GraphQL for any frontend. Supports media management via MinIO.

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
- `cms_content_types`, content type definitions
- `cms_posts`, content items
- `cms_post_versions`, version history
- `cms_categories`, category tree
- `cms_tags`, tag definitions
- `cms_post_categories`, post-category relationships
- `cms_post_tags`, post-tag relationships
- `cms_webhook_events`, webhook event log

## Security

All CMS content is protected by **Hasura row-level security (RLS)** using the `source_account_id` column. This ensures that content created in one app cannot be read or modified by other apps in the same deployment.

- Multi-app isolation: each app has its own content silo
- User-level filtering: users can only access content for their app
- Token-based access control: JWT tokens determine accessible data

For detailed RLS configuration, see the [CMS plugin source documentation](https://github.com/nself-org/plugins-pro).

## Nginx Routes

| Route | Target |
|-------|--------|
| `/cms/` | CMS management API |
| `/cms/admin` | CMS admin UI |
