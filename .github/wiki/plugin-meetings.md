# Meetings Plugin

> Room booking, recurring meeting management, calendar sharing, participant reminders, and waitlist. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install meetings
```

## What It Does

The meetings plugin provides a full room booking system with support for recurring meetings, shared calendars, and per-participant reminders. When a room reaches capacity, latecomers are added to an automatic waitlist and promoted when a slot opens, with all state changes written to an immutable audit log.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MEETINGS_PORT` | `3130` | Port the service listens on |
| `MEETINGS_DEFAULT_DURATION` | `60` | Default meeting duration in minutes |

## Ports

| Port | Purpose |
|------|---------|
| `3130` | Meetings REST API |

## Database Tables

9 tables added to your Postgres database:

- `np_meetings_rooms`
- `np_meetings_bookings`
- `np_meetings_recurring`
- `np_meetings_participants`
- `np_meetings_reminders`
- `np_meetings_waitlist`
- `np_meetings_shares`
- `np_meetings_notifications`
- `np_meetings_audit`

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/meetings/` | `localhost:3130` |
