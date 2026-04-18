# Chat Plugin

> Real-time chat with rooms, direct messages, reactions, and moderation. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install chat
```

## What It Does

Adds a complete real-time chat backend to your nSelf stack. Supports group rooms, direct messages between users, message reactions, thread replies, and built-in moderation tools. Powers the [nSelf Chat](https://github.com/nself-org/nchat) open-source client app. Exposes a REST API and WebSocket connections for real-time message delivery.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CHAT_PORT` | `3401` | Chat service port |
| `CHAT_MAX_MESSAGE_LENGTH` | `4096` | Max characters per message |
| `CHAT_FILE_UPLOAD_ENABLED` | `true` | Allow file attachments |
| `CHAT_MODERATION_ENABLED` | `true` | Enable content moderation |
| `CHAT_HISTORY_RETENTION_DAYS` | `365` | Days to retain messages |

## Ports

| Port | Purpose |
|------|---------|
| 3401 | Chat REST API and WebSocket |

## Database Tables

6 tables added to your Postgres database:
- `np_chat_rooms` — room definitions and settings
- `np_chat_participants` — room membership
- `np_chat_messages` — message content and metadata
- `np_chat_reactions` — message reactions
- `np_chat_threads` — threaded reply chains
- `np_chat_moderation` — moderation actions and bans

## Nginx Routes

| Route | Target |
|-------|--------|
| `/chat/` | Chat REST API |
| `/chat/ws` | WebSocket endpoint |
