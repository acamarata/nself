# Feature: nChat

nChat is an open-source self-hosted messaging application built on nSelf. It provides real-time team and personal chat with video calls, bots, moderation, and white-label support.

**Status:** Active
**Repo:** nself-org/chat (Type C reference app)
**Marketing site:** chat.nself.org (web/chat/)
**Platforms:** Web (Next.js 15). Mobile (Capacitor) and Desktop (Electron, Tauri) are scaffolded but not shipped.

---

## What nChat Does

nChat is a self-hosted Slack/Discord alternative. You run it on your own nSelf server, and your team connects via web browser or (eventually) native apps. All messages, files, and call recordings stay on your hardware.

### Current Features

- **Real-time messaging.** Powered by Hasura GraphQL subscriptions. Messages appear instantly across all connected clients.
- **Multi-workspace support.** Run multiple isolated workspaces on a single server. Each workspace has its own channels, members, and settings.
- **White-label configuration.** AppConfig-driven branding. Change name, logo, colors, and domain for each workspace.
- **8 test users (dev mode).** FauxAuth provides instant login for development and testing without real authentication.
- **Production auth.** Nhost authentication with OAuth providers, email/password, and MFA.
- **Web platform.** Next.js 15 application with responsive design.

### With Pro Plugin Bundle

The nChat plugin bundle adds:

| Plugin | What It Adds |
|--------|-------------|
| `chat` | Channel types (public, private, DM), typing indicators, read receipts, reactions, threads |
| `livekit` | Video and audio calls (WebRTC via LiveKit) |
| `recording` | Call recording and playback |
| `moderation` | Content filtering, user reports, admin actions |
| `bots` | Bot framework for automated responses and integrations |
| `realtime` | Enhanced WebSocket support for presence and live updates |
| `auth` | Advanced auth features (SSO, SAML, LDAP integration) |

---

## Architecture

nChat follows the standard nSelf reference app pattern:

```
┌──────────────────────────────────┐
│  nChat Client (Next.js 15)       │
│  - Channels, DMs, threads        │
│  - Video calls (LiveKit)         │
│  - File sharing                  │
│  - Emoji reactions               │
└──────────┬───────────────────────┘
           │ GraphQL subscriptions + REST
           ▼
┌──────────────────────────────────┐
│  nSelf Backend                    │
│  - Hasura (messages, channels)   │
│  - Auth (users, sessions)        │
│  - MinIO (file attachments)      │
│  - nChat plugins (chat, livekit) │
└──────────────────────────────────┘
```

The client connects to Hasura for all data operations. Real-time updates use GraphQL subscriptions over WebSocket. File uploads go through MinIO. Video calls use LiveKit's WebRTC infrastructure, self-hosted on your server.

---

## Setup

```bash
# Basic setup (free plugins only)
nself init
nself build && nself start
# nChat web app connects to Hasura directly

# With Pro plugins
nself license set nself_pro_xxxxx...
nself plugin install chat livekit recording moderation bots realtime auth
nself build && nself start
```

---

## Distinction: chat/ vs chat.nself.org

These are two different things:

| | chat/ repo | chat.nself.org |
|---|-----------|---------------|
| **What** | Open-source chat client app | Marketing website |
| **Code** | nself-org/chat | web/chat/ (in web monorepo) |
| **Hosted by** | Users, on their own servers | nSelf, on nself.org |
| **Purpose** | Production messaging app | Feature overview, download links |

---

## Pricing

nChat itself is free and open-source (MIT). The nChat plugin bundle for advanced features costs $0.99/mo or $9.99/yr. The ɳSelf+ subscription ($49/yr) includes the nChat bundle along with everything else.

---

## Planned Features

- **Mobile apps.** Capacitor (iOS/Android) shells are scaffolded. Native push notifications planned.
- **Desktop apps.** Electron and Tauri shells are scaffolded.
- **React Native.** Not started. Under consideration for native mobile experience.
- **End-to-end encryption.** Planned for DMs and private channels.
- **Federation.** Planned exploration of Matrix protocol compatibility for cross-server messaging.

---

## Related Pages

- [[Plugin-Licensing]] -- bundle pricing
- [[Feature-Auth]] -- authentication setup
- [[Feature-Storage]] -- MinIO for file attachments
- [[Feature-Plugins]] -- plugin system overview

---

← [[Features]] | [[_Sidebar]]
