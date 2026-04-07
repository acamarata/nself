# Plugin Overview

## Contents

- [Plugin Tiers](#plugin-tiers)
- [Plugin Categories](#plugin-categories)
- [How Plugins Work](#how-plugins-work)
- [Related Pages](#related-pages)

nSelf plugins extend your backend stack with additional services and capabilities. Install them with a single command — no manual Docker configuration required.

## Plugin Tiers

| Tier | Price | Includes |
|------|-------|---------|
| **Free** | $0 | Core CLI + 25 free plugins (MIT licensed) |
| **Basic** | $0.99/mo or $9.99/yr | All 62 pro plugins |
| **Pro** | $1.99/mo or $19.99/yr | Basic + AI suite (ai, claw, mux, voice, browser) |
| **Elite** | $4.99/mo or $49.99/yr | Pro + email support |
| **Business** | $9.99/mo or $99.99/yr | Elite + 24h email support + priority feature requests |
| **Business+** | $49.99/mo or $499.99/yr | Business + dedicated support channel |
| **Enterprise** | $99.99/mo or $999.99/yr | Business+ + managed DevOps |

Annual plans save ~17% compared to monthly. Existing $9.99/yr keys are grandfathered to the Basic tier.

## Plugin Categories

### Free Plugins (25) — MIT Licensed

No license key required. Install with `nself plugin install {name}`.

| Category | Plugins |
|----------|---------|
| Background Jobs | jobs, cron |
| Search | search |
| Feature Management | feature-flags, webhooks |
| Identity | invitations, tokens |
| Integrations | github, github-runner, donorbox, paypal, shopify, stripe |
| Content | content-acquisition, content-progress, torrent-manager, subtitle-manager |
| Network | vpn, mdns, link-preview |
| Notifications | notifications, notify |
| Infrastructure | backup, monitoring, mlflow |

### Pro Plugins (62) — License Key Required

Require a valid nSelf license key. See [[Plugin-Licensing]].

| Category | Count | Plugins |
|----------|-------|---------|
| AI & Automation | 7 | ai, claw, claw-web, claw-budget, mux, workflows, bots |
| Communication | 6 | chat, livekit, streaming, voice, podcast, realtime |
| Media & Processing | 6 | media-processing, file-processing, epg, photos, recording, stream-gateway |
| Content & Social | 8 | cms, social, activity-feed, moderation, knowledge-base, support, documents, calendar |
| Commerce & Payments | 4 | donorbox-pro, entitlements, analytics, backup-pro |
| Authentication & Security | 5 | auth, access-controls, idme, compliance, admin-api |
| Infrastructure | 8 | browser, google, cloudflare, object-storage, cdn, observability, ddns, devices |
| Productivity | 2 | cron-pro, notify-pro |
| Gaming & Specialized | 9 | tmdb, game-metadata, retro-gaming, rom-discovery, sports, geocoding, geolocation, home, web3 |
| Social Automation | 2 | post, meetings |

## How Plugins Work

Plugins inject into your nSelf stack via Docker Compose overlays. When you run `nself build`, all installed plugins are merged into a single `docker-compose.yml`. Each plugin:

- Runs as its own container(s) on a dedicated port
- Gets its own Postgres schema (`np_{name}`) with isolated tables
- Injects nginx routes automatically
- Declares environment variables in `plugin.json`

After installing any plugin, run `nself build` to regenerate your compose file.

## Plugin Bundles

Bundles are groups of plugins that work together to provide a complete feature set.

| Bundle | Plugins | Use Case |
|--------|---------|----------|
| [nMedia](bundle-nmedia) | 12 plugins (4 free + 8 pro) | Self-hosted media acquisition, processing, and streaming |
| ɳClaw | 13 plugins | AI personal assistant |
| nChat | 7 plugins | Messaging with video calls and moderation |

## Related Pages

- [[Plugin-Install]] — How to install, remove, and update plugins
- [[Plugin-Licensing]] — License keys, tiers, and validation
- [[Plugin-Dev-Guide]] — Build your own plugin
- [[Plugin-Architecture]] — Technical internals
- [[bundle-nmedia]] — nMedia plugin bundle guide

---
← [[Home]] | [[_Sidebar]]
