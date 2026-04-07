# Feature: nFamily

nFamily is a planned private social media platform for families, built on nSelf. It provides a self-hosted space for family members to share photos, updates, and conversations without any data leaving your own server.

**Status:** Planned (not in current development scope)
**Repo:** TBD
**Relationship to Flock:** Independent. nFamily is an nSelf product. Flock is a Unity (unyeco) product. They share no code or infrastructure. The privacy boundary between the two projects is intentional.

---

## Concept

Most families share photos and updates across a mix of iMessage, WhatsApp, Instagram, and Google Photos. None of these are private. None of them give you ownership of your data. nFamily is a self-hosted alternative where:

- **You own the server.** All photos, posts, and messages stay on hardware you control.
- **No ads, no algorithms.** The feed is chronological. No engagement optimization.
- **Invitation only.** Family members join via invite codes. No public profiles, no discoverability.
- **Multi-generational.** Simple enough for grandparents. Parental controls for kids.
- **One subscription.** The nFamily plugin bundle runs on your existing nSelf server. No per-user pricing.

---

## Planned Features

### Photo Sharing
- Upload photos and albums from any device
- Automatic organization by date and location
- Face detection and grouping (local processing, no cloud)
- Shared albums with comments and reactions
- Original quality preservation (no compression)

### Activity Feed
- Chronological timeline of family activity
- Posts with text, photos, and location
- Reactions and threaded comments
- Birthday and anniversary reminders

### Messaging
- Real-time family group chat
- Direct messages between family members
- Photo and video sharing in chat
- Voice messages

### Content Moderation
- Parental controls per family member
- Content filtering for younger members
- Activity reports for parents
- Screen time awareness (not enforcement, that is the device's job)

### Privacy
- All data stored on your server. Zero cloud dependencies.
- No analytics, no tracking, no telemetry
- End-to-end encryption for direct messages (planned)
- Granular sharing controls (who sees what)

---

## nFamily Plugin Bundle

The nFamily bundle uses these nSelf Pro plugins:

| Plugin | Purpose |
|--------|---------|
| `social` | Activity feed, posts, reactions, follows |
| `photos` | Photo upload, organization, face detection |
| `activity-feed` | Timeline aggregation and display |
| `moderation` | Content filtering and parental controls |
| `realtime` | WebSocket connections for live updates |
| `cms` | Content management for family pages/blogs |
| `chat` | Real-time messaging |

These plugins are shared with other nSelf products. The nFamily client app combines them into a family-specific experience.

---

## Architecture

nFamily follows the same pattern as other nSelf reference apps:

1. **Flutter client app** (iOS, Android, Web) communicates with the nSelf backend via Hasura GraphQL.
2. **nSelf backend** runs the nFamily plugin bundle. Plugins handle photo processing, feed generation, real-time messaging, and moderation.
3. **MinIO** stores photos and media at original quality.
4. **Auth** handles family member accounts, invitations, and role-based access.

No dedicated server is required. nFamily runs alongside your other nSelf apps on the same server.

---

## Pricing

nFamily will be available as a plugin bundle at $0.99/mo or $9.99/yr. The ɳSelf+ subscription ($49/yr) includes nFamily along with all other plugin bundles.

---

## Relationship to Flock

Flock (unyeco/flock) is a separate family app built on Unity infrastructure. nFamily and Flock are completely independent:

| | nFamily | Flock |
|---|---------|-------|
| **Organization** | nSelf (nself-org) | Unity (unyeco) |
| **Backend** | nSelf CLI stack | Unity shared backend |
| **Hosting** | Self-hosted by user | Unity servers |
| **Data ownership** | User owns everything | Unity-managed |
| **Target** | Privacy-first families | Social families |
| **Status** | Planned | Active |

The separation is by design. nFamily serves users who want complete data ownership. Flock serves users who prefer a managed experience.

---

## Related Pages

- [[Feature-Plugins]] -- plugin system overview
- [[Plugin-Licensing]] -- bundle pricing

---

← [[Features]] | [[_Sidebar]]
