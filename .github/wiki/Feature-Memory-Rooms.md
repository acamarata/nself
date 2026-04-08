# Feature: Memory Rooms

Memory Rooms let you organize nClaw's knowledge into named collections. Think of them as folders for your AI's memory. You can drag and drop memories between rooms, and each room tracks its own health metrics.

**Route:** `/settings/memory` in claw-web
**API:** Memory rooms CRUD under `/claw/memory/rooms`

---

## How It Works

Every memory in nClaw belongs to a room. The default room holds unorganized memories. You create named rooms for topics you care about (work, health, projects, people) and assign memories to them.

The room view shows:
- Memory count per room
- Health score (based on freshness, connections, usage)
- Drag-and-drop reordering
- Bulk assignment of memories

## API

### List Rooms

```
GET /claw/memory/rooms
Authorization: Bearer {token}
```

### Create Room

```
POST /claw/memory/rooms
Content-Type: application/json
Authorization: Bearer {token}

{
  "name": "Work Projects",
  "description": "Memories about active work projects"
}
```

### Assign Memories

```
POST /claw/memory/rooms/{id}/assign
Content-Type: application/json

{
  "memory_ids": ["mem_abc", "mem_def"]
}
```

### Update Room

```
PUT /claw/memory/rooms/{id}
```

### Delete Room

```
DELETE /claw/memory/rooms/{id}
```

Deleting a room moves its memories back to the default room.

## Knowledge Health

The `/knowledge` route shows a brain health report across all rooms. It checks for:
- Stale memories (not accessed in 30+ days)
- Orphan memories (no connections to other knowledge)
- Duplicate or near-duplicate entries
- Low-confidence facts that need verification

The health check runs via `GET /claw/knowledge/health`.

## Ingestion Pipeline

The auto-ingestion system imports knowledge from external sources:
- URLs (web pages, articles)
- Files (PDF, Markdown, text)
- Clipboard content

Monitor progress at `/settings/knowledge/ingestion`. The ingestion queue is backed by `np_claw.ingestion_queue`.

## Export

Export your knowledge graph in two formats:
- **Obsidian** -- Markdown files with wiki-style links
- **JSON** -- Full structured export

`GET /claw/knowledge/export?format=obsidian`

## Related

- [[Feature-nClaw]] -- nClaw overview
- [[Feature-Agent-Dashboard]] -- Agent metrics
- [[Feature-Image-Generation]] -- Image generation
