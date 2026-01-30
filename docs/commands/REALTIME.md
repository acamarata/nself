# nself realtime - Real-Time Communication

Real-time communication system for WebSocket channels, presence tracking, broadcast messaging, and database subscriptions (Change Data Capture).

**Version:** v0.9.5+
**Status:** Production Ready

---

## Synopsis

```bash
nself realtime <command> [options]
```

---

## Description

The `nself realtime` command provides a complete real-time communication system compatible with Supabase and Nhost. It includes:

- **Database Subscriptions (CDC)** - Subscribe to table changes (INSERT, UPDATE, DELETE)
- **Channel Management** - Public, private, and presence channels
- **Broadcast Messages** - Send messages to channel subscribers
- **Presence Tracking** - Track user online/away/offline status
- **WebSocket Server** - Automatic reconnection and connection pooling

---

## System Management

### init

Initialize the real-time system and database schema.

```bash
nself realtime init
```

**What it does:**
- Creates real-time database schema and tables
- Sets up WebSocket server configuration
- Initializes presence tracking system
- Creates default channels

**Example:**
```bash
nself realtime init
```

---

### status

Show real-time system status.

```bash
nself realtime status
```

**Shows:**
- WebSocket server status
- Active connections count
- Enabled channels count
- Subscription count
- System configuration

**Example:**
```bash
nself realtime status
```

**Output:**
```
Real-Time System Status

✓ WebSocket server running
✓ 145 active connections
✓ 23 channels (12 public, 8 private, 3 presence)
✓ 47 active subscriptions
✓ Presence tracking enabled
```

---

### logs

View real-time system logs.

```bash
nself realtime logs [--follow]
```

**Options:**
- `--follow` - Stream logs in real-time

**Examples:**
```bash
# View recent logs
nself realtime logs

# Follow logs
nself realtime logs --follow
```

---

### cleanup

Clean up stale connections and old messages.

```bash
nself realtime cleanup
```

**What it cleans:**
- Stale WebSocket connections (timeout: 5 minutes)
- Old messages (default retention: 30 days)
- Expired presence records
- Orphaned subscriptions

**Example:**
```bash
nself realtime cleanup
```

---

## Database Subscriptions (CDC)

Subscribe to database table changes using PostgreSQL Change Data Capture (CDC).

### subscribe

Subscribe to table changes.

```bash
nself realtime subscribe <table> [events]
```

**Arguments:**
- `<table>` - Table name (schema.table format)
- `[events]` - Comma-separated events: INSERT,UPDATE,DELETE (default: all)

**Examples:**
```bash
# Subscribe to all events
nself realtime subscribe public.users

# Subscribe to specific events
nself realtime subscribe public.posts INSERT,UPDATE

# Subscribe to schema
nself realtime subscribe public.*
```

---

### unsubscribe

Remove subscription from table.

```bash
nself realtime unsubscribe <table>
```

**Example:**
```bash
nself realtime unsubscribe public.users
```

---

### listen

Listen to table changes in real-time (blocking).

```bash
nself realtime listen <table> [seconds]
```

**Arguments:**
- `<table>` - Table to listen to
- `[seconds]` - Duration to listen (default: 60)

**Example:**
```bash
# Listen for 60 seconds
nself realtime listen public.users

# Listen for 5 minutes
nself realtime listen public.users 300
```

**Output:**
```
Listening to public.users...
[2026-01-30 10:15:23] INSERT: {"id": 123, "email": "user@example.com"}
[2026-01-30 10:15:45] UPDATE: {"id": 123, "name": "John Doe"}
```

---

### subscriptions

List all active subscriptions.

```bash
nself realtime subscriptions
```

**Example:**
```bash
nself realtime subscriptions
```

**Output:**
```
Active Database Subscriptions:

  public.users              INSERT, UPDATE, DELETE
  public.posts              INSERT, UPDATE
  public.comments           INSERT, DELETE
```

---

## Channel Management

Create and manage real-time channels for messaging.

### channel create

Create a new channel.

```bash
nself realtime channel create <name> [type]
```

**Arguments:**
- `<name>` - Channel name
- `[type]` - Channel type: public|private|presence (default: public)

**Channel Types:**
- **public** - Open to all users
- **private** - Invite-only, requires authorization
- **presence** - Tracks online users

**Examples:**
```bash
# Public channel (anyone can join)
nself realtime channel create general public

# Private channel (invite only)
nself realtime channel create team-alpha private

# Presence channel (tracks online users)
nself realtime channel create lobby presence
```

---

### channel list

List all channels.

```bash
nself realtime channel list [type]
```

**Arguments:**
- `[type]` - Filter by type: all|public|private|presence (default: all)

**Examples:**
```bash
# All channels
nself realtime channel list

# Only public channels
nself realtime channel list public

# Only presence channels
nself realtime channel list presence
```

**Output:**
```
Channels:

  general          public      245 members
  announcements    public      1,203 members
  team-alpha       private     12 members
  lobby            presence    89 online
```

---

### channel get

Get channel details.

```bash
nself realtime channel get <id>
```

**Example:**
```bash
nself realtime channel get general
```

**Output:**
```
Channel: general
Type: public
Members: 245
Created: 2026-01-15 08:30:00
Messages (24h): 1,847
```

---

### channel delete

Delete a channel.

```bash
nself realtime channel delete <id>
```

**Warning:** This deletes all channel messages and memberships.

**Example:**
```bash
nself realtime channel delete old-channel
```

---

### channel members

List channel members.

```bash
nself realtime channel members <id>
```

**Example:**
```bash
nself realtime channel members general
```

**Output:**
```
Members of #general (245):

  user-123    John Doe       online
  user-456    Jane Smith     away
  user-789    Bob Johnson    offline
```

---

### channel join

Add user to channel.

```bash
nself realtime channel join <channel> <user>
```

**Example:**
```bash
nself realtime channel join general user-123
```

---

### channel leave

Remove user from channel.

```bash
nself realtime channel leave <channel> <user>
```

**Example:**
```bash
nself realtime channel leave general user-123
```

---

## Broadcast Messages

Send messages to channel subscribers.

### broadcast

Send message to channel.

```bash
nself realtime broadcast <channel> <event> <payload>
```

**Arguments:**
- `<channel>` - Channel name or ID
- `<event>` - Event type (e.g., user.joined, message.new)
- `<payload>` - JSON payload

**Examples:**
```bash
# User joined event
nself realtime broadcast general user.joined '{"user_id": "123", "name": "John"}'

# New message event
nself realtime broadcast general message.new '{"text": "Hello!", "from": "user-123"}'

# Custom event
nself realtime broadcast general app.notification '{"title": "Update", "body": "New version available"}'
```

---

### messages

Get recent channel messages.

```bash
nself realtime messages <channel> [limit]
```

**Arguments:**
- `<channel>` - Channel name
- `[limit]` - Number of messages (default: 50, max: 1000)

**Example:**
```bash
# Last 50 messages
nself realtime messages general

# Last 100 messages
nself realtime messages general 100
```

---

### replay

Replay messages since timestamp.

```bash
nself realtime replay <channel> <timestamp>
```

**Arguments:**
- `<channel>` - Channel name
- `<timestamp>` - ISO 8601 timestamp

**Example:**
```bash
# Replay messages from today
nself realtime replay general 2026-01-30T00:00:00Z

# Replay last hour
nself realtime replay general 2026-01-30T09:00:00Z
```

---

### events

List event types in channel.

```bash
nself realtime events <channel> [hours]
```

**Arguments:**
- `<channel>` - Channel name
- `[hours]` - Timeframe in hours (default: 24)

**Example:**
```bash
# Event types in last 24 hours
nself realtime events general

# Event types in last week
nself realtime events general 168
```

**Output:**
```
Event Types (last 24h):

  user.joined       47 events
  message.new       1,203 events
  user.left         39 events
  typing.start      567 events
```

---

## Presence Tracking

Track user online/away/offline status.

### presence track

Track user presence.

```bash
nself realtime presence track <user> <channel> [status]
```

**Arguments:**
- `<user>` - User ID
- `<channel>` - Channel name
- `[status]` - Status: online|away|offline (default: online)

**Examples:**
```bash
# User comes online
nself realtime presence track user-123 general online

# User goes away
nself realtime presence track user-123 general away

# User goes offline
nself realtime presence track user-123 general offline
```

---

### presence get

Get user presence.

```bash
nself realtime presence get <user> [channel]
```

**Arguments:**
- `<user>` - User ID
- `[channel]` - Channel name (optional, shows all if omitted)

**Examples:**
```bash
# All channels
nself realtime presence get user-123

# Specific channel
nself realtime presence get user-123 general
```

**Output:**
```
Presence: user-123

  general          online     Last seen: now
  team-alpha       away       Last seen: 5m ago
  support          offline    Last seen: 2h ago
```

---

### presence online

List online users.

```bash
nself realtime presence online [channel]
```

**Arguments:**
- `[channel]` - Channel name (optional, shows global if omitted)

**Examples:**
```bash
# Global online users
nself realtime presence online

# Channel online users
nself realtime presence online general
```

**Output:**
```
Online Users (general): 89

  user-123    John Doe       online    5s ago
  user-456    Jane Smith     online    12s ago
  user-789    Bob Johnson    away      2m ago
```

---

### presence count

Count online users.

```bash
nself realtime presence count [channel]
```

**Example:**
```bash
# Global count
nself realtime presence count

# Channel count
nself realtime presence count general
```

**Output:**
```
Online: 89
Away: 12
Offline: 144
Total: 245
```

---

### presence offline

Set user offline.

```bash
nself realtime presence offline <user> [channel]
```

**Examples:**
```bash
# Offline in all channels
nself realtime presence offline user-123

# Offline in specific channel
nself realtime presence offline user-123 general
```

---

### presence stats

Get presence statistics.

```bash
nself realtime presence stats
```

**Example:**
```bash
nself realtime presence stats
```

**Output:**
```
Presence Statistics:

  Total users tracked:     1,247
  Currently online:        389
  Currently away:          78
  Currently offline:       780

  Channels with presence:  15
  Average presence/channel: 83
```

---

## Connection Management

### connections

Show active WebSocket connections.

```bash
nself realtime connections [--json]
```

**Options:**
- `--json` - Output in JSON format

**Example:**
```bash
nself realtime connections
```

**Output:**
```
Active Connections: 145

  conn-abc123    user-123    general, team-alpha     Connected 5m ago
  conn-def456    user-456    general                 Connected 12m ago
  conn-ghi789    user-789    support                 Connected 1h ago
```

---

### stats

Show detailed real-time statistics.

```bash
nself realtime stats
```

**Example:**
```bash
nself realtime stats
```

**Output:**
```
Real-Time Statistics:

WebSocket Connections:
  Active:                  145
  Peak (24h):             287
  Average latency:         15ms

Channels:
  Total:                   23
  Public:                  12
  Private:                 8
  Presence:                3

Messages (24h):
  Sent:                    45,203
  Delivered:               45,187
  Failed:                  16

Database Subscriptions:
  Active:                  47
  Tables monitored:        12
  Events processed (24h):  8,934

Presence:
  Users tracked:           1,247
  Currently online:        389
  Heartbeats (1m):         389
```

---

## Configuration

Real-time system configuration via environment variables:

```bash
# .env
REALTIME_ENABLED=true
REALTIME_PORT=4000
REALTIME_MAX_CONNECTIONS=10000
REALTIME_MESSAGE_TTL=86400          # 1 day
REALTIME_PRESENCE_TIMEOUT=300       # 5 minutes
REALTIME_HEARTBEAT_INTERVAL=30      # 30 seconds
REALTIME_RECONNECT_DELAY=5000       # 5 seconds
```

---

## Database Schema

Real-time system uses these PostgreSQL tables:

```sql
-- Channels
realtime.channels (id, name, type, created_at, metadata)

-- Channel members
realtime.channel_members (channel_id, user_id, joined_at)

-- Messages
realtime.messages (id, channel_id, event, payload, created_at)

-- Presence
realtime.presence (user_id, channel_id, status, last_seen, metadata)

-- Subscriptions
realtime.subscriptions (id, table_name, events, created_at)

-- Connections
realtime.connections (id, user_id, connected_at, last_ping)
```

---

## Examples

### Complete Setup

```bash
# 1. Initialize system
nself realtime init

# 2. Create channels
nself realtime channel create general public
nself realtime channel create announcements public
nself realtime channel create support private

# 3. Subscribe to database changes
nself realtime subscribe public.users INSERT,UPDATE,DELETE
nself realtime subscribe public.posts INSERT,UPDATE

# 4. Check status
nself realtime status
```

### User Joins Channel

```bash
# Add user to channel
nself realtime channel join general user-123

# Track presence
nself realtime presence track user-123 general online

# Broadcast join event
nself realtime broadcast general user.joined '{"user_id": "user-123", "name": "John Doe"}'
```

### Monitor Activity

```bash
# Watch database changes
nself realtime listen public.users 300

# View channel messages
nself realtime messages general 50

# Check who's online
nself realtime presence online general

# View statistics
nself realtime stats
```

---

## Performance

- **WebSocket Connections:** 10,000+ concurrent per instance
- **Message Delivery:** <10ms latency
- **Presence Update:** <50ms propagation
- **Database CDC:** <20ms event delivery
- **Channel Broadcast:** <30ms to all subscribers

---

## Security

### Authentication

All real-time operations require authentication via JWT tokens.

### Authorization

- **Public channels:** Anyone can join
- **Private channels:** Requires membership
- **Presence channels:** Membership + presence tracking

### Rate Limiting

- **Connections:** 100 per IP per minute
- **Messages:** 60 per connection per minute
- **Presence updates:** 10 per connection per minute

---

## Troubleshooting

### Connection Issues

```bash
# Check WebSocket server
nself realtime status

# View logs
nself realtime logs --follow

# Check connections
nself realtime connections
```

### Message Delivery Issues

```bash
# Verify channel exists
nself realtime channel list

# Check recent messages
nself realtime messages <channel> 10

# Monitor broadcast
nself realtime listen <table>
```

### Presence Not Updating

```bash
# Check presence configuration
nself realtime presence stats

# Manually update presence
nself realtime presence track <user> <channel> online

# Clean up stale presence
nself realtime cleanup
```

---

## Migration from Supabase

ɳSelf real-time is compatible with Supabase real-time API:

```bash
# Import Supabase real-time configuration
nself migrate from-supabase realtime

# Channels are automatically created
# Subscriptions are migrated
# Presence tracking continues working
```

---

## See Also

- [Real-Time Features Guide](../guides/REALTIME-FEATURES.md)
- [WebSocket API Reference](../api/WEBSOCKET-API.md)
- [Database Subscriptions](../guides/DATABASE-SUBSCRIPTIONS.md)
- [Presence Tracking](../guides/PRESENCE-TRACKING.md)
- [nself status](./STATUS.md)

---

**Version:** ɳSelf v0.9.5
**Last Updated:** January 30, 2026
