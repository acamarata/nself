# Social Plugin

> Social features — posts, comments, reactions, follows, and bookmarks. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install social
```

## What It Does

Adds a complete social layer to your application. Users can post content, comment on posts, react with emoji, follow other users, and bookmark content. All social data is stored in Postgres and tracked in Hasura GraphQL. Integrates with the `activity-feed` plugin for personalized feed generation.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SOCIAL_PORT` | `3502` | Social service port |
| `SOCIAL_MAX_POST_LENGTH` | `2000` | Max characters per post |
| `SOCIAL_REACTIONS_ENABLED` | `true` | Enable reaction system |
| `SOCIAL_MODERATION_ENABLED` | `true` | Enable content moderation |

## Ports

| Port | Purpose |
|------|---------|
| 3502 | Social REST API |

## Database Tables

7 tables added to your Postgres database:
- `np_social_posts` — post content
- `np_social_comments` — comment threads
- `np_social_reactions` — emoji reactions
- `np_social_follows` — follow relationships
- `np_social_bookmarks` — saved/bookmarked content
- `np_social_shares` — reshare records
- `np_social_mentions` — @mention records

## Nginx Routes

| Route | Target |
|-------|--------|
| `/social/` | Social REST API |
