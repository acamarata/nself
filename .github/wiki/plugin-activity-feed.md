# Activity Feed Plugin

> Fan-out activity feed with aggregation and real-time subscriptions. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install activity-feed
```

## What It Does

Generates personalized activity feeds by fanning out social events to follower feeds. When a user posts, their followers' feeds are updated immediately. Groups similar activities to reduce noise (e.g., "5 people liked your post" instead of 5 separate items). Provides a subscription API for real-time feed updates. Integrates with the `social` plugin.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ACTIVITY_FEED_PORT` | `3209` | Activity feed service port |
| `ACTIVITY_FEED_AGGREGATION_WINDOW` | `3600` | Aggregation window in seconds |
| `ACTIVITY_FEED_MAX_ITEMS` | `500` | Max items per user feed |
| `ACTIVITY_FEED_TRIM_INTERVAL` | `3600` | Feed trim interval in seconds |

## Ports

| Port | Purpose |
|------|---------|
| 3209 | Activity feed REST API |

## Database Tables

4 tables added to your Postgres database:
- `np_activity_feed_activities` — activity event log
- `np_activity_feed_feeds` — per-user feed items
- `np_activity_feed_aggregations` — grouped activity aggregations
- `np_activity_feed_subscriptions` — real-time subscription records

## Nginx Routes

| Route | Target |
|-------|--------|
| `/activity-feed/` | Activity feed API |
