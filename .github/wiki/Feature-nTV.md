# Feature: ɳTV

ɳTV is an open-source media player and server built on ɳSelf. It runs on six platforms (iOS, Android, macOS, Windows, Linux, Web) using Flutter and connects to a self-hosted ɳSelf backend for media management, streaming, and metadata.

**Status:** Planned
**Repo:** nself-org/ntv (Type C reference app)
**Marketing site:** ntv.nself.org (web/ntv/)

---

## What ɳTV Does

ɳTV turns your ɳSelf server into a personal media system. You store your media library on your own hardware, and ɳTV clients on any device play it back with transcoding, metadata, and EPG support. Think Jellyfin or Plex, but built on the ɳSelf stack so it shares authentication, storage, and infrastructure with your other ɳSelf apps.

Key capabilities:

- **Library management.** Scan, organize, and browse movies, TV shows, music, and photos stored on your server.
- **Adaptive streaming.** Server-side transcoding delivers the right quality for your bandwidth and device. HLS and DASH protocols.
- **Metadata enrichment.** Automatic lookup from TMDB, MusicBrainz, and other sources. Poster art, descriptions, cast, ratings.
- **EPG (Electronic Program Guide).** For IPTV and live TV sources. Schedule display, recording, and time-shifting.
- **Multi-user.** Each household member gets their own profile, watch history, and recommendations. Parental controls per profile.
- **Offline playback.** Download media to your device for offline viewing. Sync state tracks what you have downloaded.
- **Subtitle management.** Auto-download subtitles from OpenSubtitles and other sources. Multi-language support.

---

## nMedia Plugin Bundle

ɳTV requires the **nMedia** plugin bundle on the server side. nMedia is a marketing name for a collection of 12 plugins in the ɳSelf plugin ecosystem. It is not a separate repository.

### Plugins in the nMedia Bundle

**Free plugins (4):**

| Plugin | Purpose |
|--------|---------|
| `torrent-manager` | BitTorrent client integration for media acquisition |
| `content-acquisition` | Automated content discovery and download orchestration |
| `content-progress` | Track download and processing progress |
| `subtitle-manager` | Subtitle search, download, and sync |

**Pro plugins (8):**

| Plugin | Purpose |
|--------|---------|
| `media-processing` | Transcoding, format conversion, thumbnail generation |
| `file-processing` | File organization, deduplication, metadata extraction |
| `streaming` | HLS/DASH adaptive streaming server |
| `stream-gateway` | Stream routing and access control |
| `epg` | Electronic Program Guide for live TV sources |
| `tmdb` | TMDB metadata lookup and enrichment |
| `recording` | DVR-style recording from live streams |
| `transcoder` | Hardware-accelerated video transcoding (NVENC, VAAPI, QSV) |

Additional media-adjacent Pro plugins that work with ɳTV but are not part of the core bundle:

| Plugin | Purpose |
|--------|---------|
| `thumb` | Thumbnail sprite generation for video scrubbing |
| `watermark` | Dynamic watermarking for shared content |
| `drm` | DRM wrapping for protected content distribution |
| `game-metadata` | Game metadata lookup (IGDB, RetroAchievements) |
| `retro-gaming` | Retro game library management and emulator integration |
| `rom-discovery` | ROM file detection and cataloging |
| `sports` | Sports schedule, scores, and event recording |

---

## Architecture

```
┌─────────────────────────────────────────────┐
│  nTV Flutter Client (6 platforms)            │
│  - Library browser                          │
│  - Video/audio player                       │
│  - Offline sync engine                      │
│  - EPG viewer                               │
└──────────────┬──────────────────────────────┘
               │ GraphQL (Hasura) + REST (plugins)
               ▼
┌─────────────────────────────────────────────┐
│  nSelf Backend                               │
│  ┌──────────────┐  ┌──────────────────────┐ │
│  │ Hasura (GQL) │  │ nMedia Plugins       │ │
│  │ - Library DB │  │ - media-processing   │ │
│  │ - User prefs │  │ - streaming          │ │
│  │ - Watch hist │  │ - epg                │ │
│  └──────────────┘  │ - tmdb               │ │
│  ┌──────────────┐  │ - transcoder         │ │
│  │ MinIO        │  │ - content-acquisition│ │
│  │ - Media files│  └──────────────────────┘ │
│  └──────────────┘                            │
└─────────────────────────────────────────────┘
```

The Flutter client communicates with Hasura for library data and user preferences. Media streaming goes through the streaming plugin, which serves HLS/DASH segments from transcoded files in MinIO or local storage. Metadata enrichment runs as background jobs via the cron plugin.

---

## Setup

ɳTV is not yet released. When available, setup will follow this pattern:

```bash
# 1. Install nMedia plugins on your nSelf server
nself license set nself_pro_xxxxx...
nself plugin install media-processing streaming epg tmdb transcoder
nself plugin install torrent-manager content-acquisition subtitle-manager
nself build && nself start

# 2. Install the nTV client on your device
# Available from App Store, Google Play, or GitHub Releases

# 3. Point nTV at your server
# Enter your server URL (e.g., https://media.yourdomain.com)
# Sign in with your nSelf auth credentials
```

---

## Pricing

ɳTV itself is free and open-source (MIT). The nMedia plugin bundle costs $0.99/mo or $9.99/yr as part of the ɳSelf Pro plugin tier. The 4 free plugins (torrent-manager, content-acquisition, content-progress, subtitle-manager) work without a license key.

---

## Related Pages

- [[Plugin-Licensing]] -- tier requirements for nMedia plugins
- [[Feature-Plugins]] -- plugin system overview
- [[Feature-Storage]] -- MinIO setup for media files

---

← [[Features]] | [[_Sidebar]]
