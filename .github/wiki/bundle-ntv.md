# ɳTV Plugin Bundle

## Contents

- [What Is ɳTV](#what-is-ntv)
- [Plugins Included](#plugins-included)
- [How They Work Together](#how-they-work-together)
- [Installation](#installation)
- [Getting Started](#getting-started)
- [Related Pages](#related-pages)

## What Is ɳTV

ɳTV is the bundle of 12 plugins that together provide a complete self-hosted media management system. It is not a repo or a single plugin. When all 12 are installed, you get a pipeline from content discovery through transcoding to streaming playback.

ɳTV pairs with the [ɳTV app](https://github.com/nself-org/ntv), a Flutter media player that connects to a ɳSelf backend running these plugins.

## Plugins Included

4 free plugins (no license needed) and 8 pro plugins ($0.99/mo or $9.99/yr).

### Free Plugins

| Plugin | Language | What It Does |
|--------|----------|-------------|
| [torrent-manager](plugin-torrent-manager) | Go | Torrent client integration with download management |
| [content-acquisition](plugin-content-acquisition) | Go | RSS monitoring, release calendars, automated download pipelines |
| [subtitle-manager](plugin-subtitle-manager) | Go | Subtitle search, download, and sync |
| [vpn](plugin-vpn) | Go | VPN/WireGuard management for secure downloading |

### Pro Plugins

| Plugin | Language | What It Does |
|--------|----------|-------------|
| [media-processing](plugin-media-processing) | Rust | Video/audio transcoding (CPU-bound) |
| [file-processing](plugin-file-processing) | Rust | File format conversion, image processing (CPU-bound) |
| [epg](plugin-epg) | Go | Electronic program guide data |
| [tmdb](plugin-tmdb) | Go | Movie/TV metadata from TMDB |
| [game-metadata](plugin-game-metadata) | Go | Game metadata and artwork lookup |
| [recording](plugin-recording) | Go | DVR recording |
| [stream-gateway](plugin-stream-gateway) | Go | HLS/DASH streaming gateway |
| [streaming](plugin-streaming) | Go | Live streaming service |

Only media-processing and file-processing use Rust. These are CPU-bound workloads where Rust provides meaningful performance gains. Everything else is Go.

## How They Work Together

```
Discovery → Acquisition → Processing → Metadata → Delivery → Recording
```

### Acquisition

1. **content-acquisition** scans RSS feeds for new content matching user-defined rules
2. **torrent-manager** downloads via a configured torrent client
3. **vpn** enforces a WireGuard tunnel for all download traffic
4. **subtitle-manager** finds and syncs matching subtitles

### Metadata

1. **tmdb** enriches movies and TV shows with posters, descriptions, ratings
2. **game-metadata** enriches games with cover art and platform info
3. **epg** provides electronic TV guide data for live and recorded content

### Processing

1. **media-processing** transcodes video and audio to target formats and quality levels
2. **file-processing** converts file formats and generates thumbnails

### Delivery

1. **stream-gateway** routes streams via HLS/DASH with adaptive bitrate
2. **streaming** provides the live streaming layer
3. **recording** handles DVR, scheduled captures, and time-shifting

## Installation

```bash
# Free plugins (no license needed)
nself plugin install torrent-manager content-acquisition subtitle-manager vpn

# Pro plugins (nTV bundle: $0.99/mo or $9.99/yr)
nself license set nself_pro_xxxxx...
nself plugin install media-processing file-processing epg tmdb game-metadata recording stream-gateway streaming

# Rebuild and start
nself build && nself start
```

## Getting Started

### Prerequisites

- ɳSelf CLI installed and a project initialized (`nself init`)
- Backend running (`nself start`)
- ɳTV bundle license ($0.99/mo or $9.99/yr)

### Step 1: Install the plugins

Install all 12 ɳTV plugins as shown above.

### Step 2: Configure media sources

Edit your `.env` to set up content-acquisition RSS feeds and torrent-manager download paths. Each plugin documents its env vars on its own wiki page.

### Step 3: Configure metadata

Add your TMDB API key to enable movie and TV metadata enrichment.

### Step 4: Set up streaming

Configure stream-gateway with your preferred HLS segment duration and adaptive bitrate profiles.

### Step 5: Connect a player

Point the ɳTV app (or any HLS/DASH-compatible player) at your ɳSelf backend's streaming endpoint.

### Troubleshooting

- **Downloads not starting:** check that vpn is running and torrent-manager can reach your torrent client
- **No metadata:** verify your TMDB API key is set in `.env`
- **Transcoding slow:** media-processing is CPU-bound. Check available CPU cores and consider hardware acceleration settings
- **Stream buffering:** adjust stream-gateway segment duration and bitrate profiles

## Related Pages

- [Plugin Overview](Plugin-Overview), all plugins and tiers
- [Plugin Install](Plugin-Install), how to install plugins
- [Plugin Licensing](Plugin-Licensing), license keys and tiers
- Individual plugin pages: [torrent-manager](plugin-torrent-manager), [content-acquisition](plugin-content-acquisition), [media-processing](plugin-media-processing), [file-processing](plugin-file-processing), [subtitle-manager](plugin-subtitle-manager), [vpn](plugin-vpn), [epg](plugin-epg), [tmdb](plugin-tmdb), [game-metadata](plugin-game-metadata), [recording](plugin-recording), [stream-gateway](plugin-stream-gateway), [streaming](plugin-streaming)
