# Blog Plugin

> Blog engine with posts, categories, tags, and SEO metadata. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install blog
```

## What It Does

A focused blog engine built on top of the CMS plugin. Adds author profiles, reading time estimates, social share metadata, RSS feed generation, and comment management. All blog content is queryable via Hasura GraphQL. Designed for marketing sites and developer blogs that need structured content and SEO control.

## Dependencies

Built on the `cms` plugin. Install cms first.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `BLOG_PORT` | `3045` | Blog service port |
| `BLOG_RSS_ENABLED` | `true` | Generate RSS/Atom feeds |
| `BLOG_COMMENTS_ENABLED` | `true` | Enable comment system |
| `BLOG_COMMENTS_MODERATION` | `true` | Require comment approval |
| `BLOG_POSTS_PER_PAGE` | `10` | Default pagination |

## Ports

| Port | Purpose |
|------|---------|
| 3045 | Blog REST API |

## Database Tables

4 tables added to your Postgres database:
- `np_blog_authors` — author profiles
- `np_blog_posts` — blog posts (extends cms posts)
- `np_blog_comments` — comment threads
- `np_blog_seo` — SEO metadata per post

## Nginx Routes

| Route | Target |
|-------|--------|
| `/blog/` | Blog API |
| `/blog/feed.xml` | RSS feed |
