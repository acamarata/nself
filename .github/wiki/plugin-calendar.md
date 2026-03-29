# Calendar Plugin

> Event management with recurring events, iCal export, RSVP, and reminders. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install calendar
```

## What It Does

A full-featured calendar backend. Create single and recurring events (daily, weekly, monthly, custom RRULE), invite users with RSVP tracking, export calendars as iCal feeds for subscription in Apple Calendar/Google Calendar, and send configurable reminders before events via the `notify` plugin.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CALENDAR_PORT` | `3104` | Calendar service port |
| `CALENDAR_DEFAULT_TIMEZONE` | `UTC` | Default event timezone |
| `CALENDAR_ICAL_ENABLED` | `true` | Enable iCal feed export |
| `CALENDAR_REMINDERS_ENABLED` | `true` | Enable event reminders |

## Ports

| Port | Purpose |
|------|---------|
| 3104 | Calendar REST API |

## Database Tables

6 tables added to your Postgres database:
- `np_calendar_calendars` — calendar definitions
- `np_calendar_events` — event records
- `np_calendar_recurrences` — RRULE recurrence definitions
- `np_calendar_attendees` — RSVP records
- `np_calendar_reminders` — reminder configurations
- `np_calendar_subscriptions` — external calendar subscriptions

## Nginx Routes

| Route | Target |
|-------|--------|
| `/calendar/` | Calendar management API |
| `/calendar/{id}/feed.ics` | iCal subscription feed |
