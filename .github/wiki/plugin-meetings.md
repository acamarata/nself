# Meetings Plugin

> Room booking, recurring calendar events, attendee RSVP, shared calendars, and availability tracking. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|----------------------|
| Free | $0 | $0 | No |
| Any bundle | $0.99/mo | $9.99/yr | Yes (pro-tier access) |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** Any paid bundle or ɳSelf+ (this is a `tier: pro` plugin per F07-PRICING-TIERS).

## Bundle membership

The meetings plugin is available with any paid bundle subscription or ɳSelf+. It is not locked to a specific bundle.

Get all bundles and all apps via **ɳSelf+** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install meetings
nself build
```

The license is validated against `ping.nself.org/license/validate`. The install will fail if no valid pro-tier license is set.

## Description

The meetings plugin provides a room booking and calendar system for nSelf deployments. It manages meeting events, attendee tracking with RSVP, recurring meetings, shared calendars, and room capacity with automatic waitlist promotion.

Rooms carry full amenity metadata (video conference, projector, whiteboard, phone, capacity). When a room reaches capacity, latecomers are queued on a waitlist and promoted automatically when a slot opens.

Meeting templates let teams predefine common event shapes (agenda, duration, room type, default attendees) so that recurring meeting patterns require no manual re-entry. Reminders are per-event and per-attendee with configurable lead times.

External calendar sync (Google Calendar, Outlook) is scaffolded at the schema level but returns HTTP 501 until OAuth providers are wired in via the `google` plugin. The sync infrastructure (external calendar records, webhook path) is in place and ready for the connection.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string (auto-provided by nself) |
| `PORT` | `3130` | Port the meetings service listens on |
| `MEETINGS_API_KEY` | — | Inter-plugin API key (auto-provided by nself) |
| `MEETINGS_RATE_LIMIT_MAX` | `100` | Max requests per rate-limit window |
| `MEETINGS_RATE_LIMIT_WINDOW_MS` | `60000` | Rate-limit window in milliseconds |
| `GOOGLE_CALENDAR_CLIENT_ID` | — | OAuth client ID for Google Calendar sync (optional) |
| `GOOGLE_CALENDAR_CLIENT_SECRET` | — | OAuth client secret for Google Calendar sync (optional) |
| `GOOGLE_CALENDAR_REDIRECT_URI` | — | OAuth redirect URI for Google Calendar (optional) |
| `OUTLOOK_CALENDAR_CLIENT_ID` | — | OAuth client ID for Outlook Calendar sync (optional) |
| `OUTLOOK_CALENDAR_CLIENT_SECRET` | — | OAuth client secret for Outlook Calendar sync (optional) |
| `OUTLOOK_CALENDAR_REDIRECT_URI` | — | OAuth redirect URI for Outlook Calendar (optional) |

## Ports

| Port | Purpose |
|------|---------|
| `3130` | Meetings REST API |

## Database Schema

9 tables added to your Postgres database under the `meetings_` prefix:

| Table | Purpose |
|-------|---------|
| `meetings_calendars` | User-owned calendar containers |
| `meetings_rooms` | Bookable rooms with amenity and capacity data |
| `meetings_events` | Meeting events with recurrence rules |
| `meetings_attendees` | Per-event attendees and RSVP state |
| `meetings_calendar_shares` | Calendar sharing grants |
| `meetings_external_calendars` | External calendar sync registrations |
| `meetings_templates` | Reusable meeting templates |
| `meetings_reminders` | Per-attendee reminder schedule |
| `meetings_waitlist` | Room capacity overflow queue |

All tables include `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation (Convention A).

## REST API

Base path: `/api/v1/`

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/calendars` | List calendars |
| `POST` | `/api/v1/calendars` | Create calendar |
| `GET` | `/api/v1/calendars/{id}` | Get calendar |
| `PUT` | `/api/v1/calendars/{id}` | Update calendar |
| `DELETE` | `/api/v1/calendars/{id}` | Delete calendar |
| `GET` | `/api/v1/calendars/{calendar_id}/shares` | List calendar shares |
| `POST` | `/api/v1/calendars/{calendar_id}/shares` | Create calendar share |
| `DELETE` | `/api/v1/calendars/{calendar_id}/shares/{id}` | Revoke calendar share |
| `GET` | `/api/v1/rooms` | List rooms |
| `POST` | `/api/v1/rooms` | Create room |
| `GET` | `/api/v1/rooms/{id}` | Get room |
| `PUT` | `/api/v1/rooms/{id}` | Update room |
| `DELETE` | `/api/v1/rooms/{id}` | Delete room |
| `GET` | `/api/v1/events` | List events |
| `POST` | `/api/v1/events` | Create event |
| `GET` | `/api/v1/events/{id}` | Get event |
| `PUT` | `/api/v1/events/{id}` | Update event |
| `DELETE` | `/api/v1/events/{id}` | Delete event |
| `GET` | `/api/v1/events/{event_id}/attendees` | List attendees |
| `POST` | `/api/v1/events/{event_id}/attendees` | Add attendee |
| `PUT` | `/api/v1/events/{event_id}/attendees/{id}/rsvp` | Update RSVP |
| `DELETE` | `/api/v1/events/{event_id}/attendees/{id}` | Remove attendee |
| `GET` | `/api/v1/events/{event_id}/reminders` | List reminders |
| `POST` | `/api/v1/events/{event_id}/reminders` | Create reminder |
| `DELETE` | `/api/v1/events/{event_id}/reminders/{id}` | Delete reminder |
| `GET` | `/api/v1/external-calendars` | List external calendar syncs |
| `POST` | `/api/v1/external-calendars` | Register external calendar |
| `DELETE` | `/api/v1/external-calendars/{id}` | Remove external calendar |
| `GET` | `/api/v1/templates` | List templates |
| `POST` | `/api/v1/templates` | Create template |
| `GET` | `/api/v1/templates/{id}` | Get template |
| `PUT` | `/api/v1/templates/{id}` | Update template |
| `DELETE` | `/api/v1/templates/{id}` | Delete template |
| `GET` | `/health` | Health check |
| `GET` | `/ready` | Readiness check |

All endpoints require `Authorization: Bearer <token>`.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/meetings/` | `localhost:3130` |

## Examples

**Create a meeting room:**

```bash
curl -X POST http://localhost:3130/api/v1/rooms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Conference A","capacity":10,"has_video_conference":true,"is_public":true}'
```

**Book a meeting event:**

```bash
curl -X POST http://localhost:3130/api/v1/events \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Sprint Planning","start_time":"2026-07-01T10:00:00Z","end_time":"2026-07-01T11:00:00Z","calendar_id":"<uuid>"}'
```

**Add an attendee with RSVP:**

```bash
curl -X POST http://localhost:3130/api/v1/events/<event_id>/attendees \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice@example.com","role":"required"}'
```

**Check plugin status:**

```bash
nself plugin meetings status
```

## Webhooks

The meetings plugin emits these events to the nself webhook bus:

| Event | Trigger |
|-------|---------|
| `event.created` | New meeting event created |
| `event.updated` | Meeting event updated |
| `event.cancelled` | Meeting event cancelled |
| `event.deleted` | Meeting event deleted |
| `rsvp.updated` | Attendee RSVP changed |
| `room.booked` | Room booking confirmed |
| `room.released` | Room booking released |
| `sync.completed` | External calendar sync finished |
| `reminder.sent` | Meeting reminder dispatched |

## Source

Source-available (license required to run): [`plugins-pro/paid/meetings/`](https://github.com/nself-org/plugins-pro/tree/main/paid/meetings)

Note: `plugins-pro` is a private repository. Source access is granted to ɳSelf+ subscribers and Enterprise customers.

## See Also

- [[plugin-calendar]] — shared family calendar (ɳFamily bundle)
- [[plugin-google]] — Google Calendar OAuth integration
- [[plugin-notify]] — push notification delivery for reminders
- [[plugin-cron]] — scheduled job support for recurring meeting tasks
- [[Plugins]] — full plugin index
- [[Pricing]] — tier comparison

← [[Plugins]] | [[Home]] →
