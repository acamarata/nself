# ɳSelf Wiki

> **Self-hosted backend in five minutes.** Postgres, GraphQL, Auth, Nginx. No cloud required.

nSelf is an open-source CLI that spins up a production-grade backend stack on any server or local machine. PostgreSQL, Hasura GraphQL, Auth, and Nginx reverse-proxy out of the box. MIT licensed, no vendor lock-in, no cloud dependency. 25 free plugins included, 52 Pro plugins available starting at $0.99/mo.

```bash
brew install nself-org/nself/nself   # or: curl -sSL https://install.nself.org | bash
nself init myapp
nself start
```

**Three commands. Complete backend. Your infrastructure.**

---

## Contents

- [Getting Started](#getting-started)
- [Core Stack](#core-stack)
- [Features](#features)
- [Commands](#commands)
- [Configuration](#configuration)
- [Plugins](#plugins)
- [Guides](#guides)
- [Architecture](#architecture)
- [Security](#security)
- [Brand](#brand)
- [Contributing](#contributing)
- [Resources](#resources)

---

## Getting Started

New to nSelf? Start here.

| Page | Description |
|------|-------------|
| [[Installation]] | Install via Homebrew, curl, or manual download (macOS, Linux, WSL) |
| [[Quick-Start]] | Zero to running backend in under 5 minutes |
| [[First-Project]] | Guided walkthrough building a real project with auth, storage, and custom services |
| [[Upgrading-from-v1]] | Migrating from the legacy Bash CLI to the Go rewrite |
| [[FAQ]] | Common questions and answers |

---

## Core Stack

Every nSelf project runs these four services. No configuration required.

| Service | Version | What it does |
|---------|---------|-------------|
| **PostgreSQL** | 16-alpine | Primary database. pgvector, PostGIS, TimescaleDB extensions available |
| **Hasura GraphQL** | v2.44.0 | Instant GraphQL API with row-level permissions, subscriptions, remote schemas |
| **Auth (nHost)** | 0.36.0 | JWT sessions, 13 OAuth providers, MFA, magic links, passwordless flows |
| **Nginx** | alpine | SSL termination, reverse proxy, rate limiting, security headers. Only external-facing service |

Optional services (enable per project): **Redis** | **MinIO** (S3 storage) | **Search** (MeiliSearch) | **Functions** | **Email** (Mailpit dev / 16+ prod providers) | **MLflow** | **Admin GUI** (localhost:3021)

:arrow_right: [[Architecture]] for full system overview, data flow, and service isolation

---

## Features

| Page | Covers |
|------|--------|
| [[Features]] | Complete feature matrix with status indicators |
| [[Feature-Auth]] | Authentication, OAuth providers, JWT, MFA, magic links |
| [[Feature-Storage]] | S3-compatible object storage with MinIO |
| [[Feature-Search]] | Full-text search with MeiliSearch |
| [[Feature-Functions]] | Serverless function runtime |
| [[Feature-Email]] | Email delivery with 16+ provider integrations |
| [[Feature-Monitoring]] | Prometheus, Grafana, Loki, Tempo (10-service monitoring bundle) |
| [[Feature-Plugins]] | Plugin system overview and capabilities |

---

## Commands

nSelf provides **32 top-level commands** with **295+ subcommands**. Full reference:

| Category | Commands | Description |
|----------|----------|-------------|
| **Lifecycle** | [[cmd-init]] · [[cmd-build]] · [[cmd-start]] · [[cmd-stop]] · [[cmd-restart]] | Create, build, and manage your stack |
| **Monitoring** | [[cmd-status]] · [[cmd-logs]] · [[cmd-health]] · [[cmd-urls]] · [[cmd-doctor]] | Observe and diagnose services |
| **Data** | [[cmd-db]] | Database operations, migrations, backup, restore |
| **Config** | [[cmd-config]] · [[cmd-service]] · [[cmd-ssl]] | Environment, services, and certificates |
| **Plugins** | [[cmd-plugin]] · [[cmd-license]] | Install, manage, and license plugins |
| **Utilities** | [[cmd-exec]] · [[cmd-clean]] · [[cmd-reset]] · [[cmd-update]] · [[cmd-version]] · [[cmd-admin]] · [[cmd-migrate]] · [[cmd-completion]] | Shell access, cleanup, updates, migrations |

:arrow_right: [[Commands]] for the full command table with descriptions

---

## Configuration

nSelf uses a layered `.env` file system. No YAML, no config files to learn.

| Page | Covers |
|------|--------|
| [[Configuration]] | Overview: env file cascade, switching environments, validation |
| [[Config-Env-Vars]] | Complete environment variable reference (every variable, every service) |
| [[Config-Postgres]] | PostgreSQL: extensions, version pinning, connection pooling |
| [[Config-Hasura]] | Hasura: admin secret, JWT, console, dev mode, metadata |
| [[Config-Auth]] | Auth: OAuth providers, SMTP, rate limiting, JWT settings |
| [[Config-Nginx]] | Nginx: domains, SSL mode, rate limits, custom locations |
| [[Config-Optional-Services]] | Redis, MinIO, Functions, Search, Email, MLflow, Admin |
| [[Config-Custom-Services]] | Custom service slots CS_1 through CS_10 with 40+ templates |
| [[Config-System]] | System-level settings: Docker, networking, ports |

---

## Plugins

77 plugins extend your stack. 25 are free (MIT), 52 require a Pro license key.

| Page | Covers |
|------|--------|
| [[Plugin-Overview]] | All plugins by category with tier requirements |
| [[Plugin-Install]] | Installing, removing, updating, and listing plugins |
| [[Plugin-Licensing]] | License keys, tiers, pricing, validation |
| [[Plugin-Architecture]] | Technical internals: compose overlays, nginx injection, schemas |
| [[Plugin-Dev-Guide]] | Build and publish your own plugin |

<details>
<summary><strong>Free Plugins (25)</strong> — MIT licensed, no key required</summary>

| Plugin | What it does |
|--------|-------------|
| [[plugin-content-acquisition]] | RSS monitoring, release calendars, download pipelines |
| [[plugin-content-progress]] | Watch/read progress tracking |
| [[plugin-feature-flags]] | Feature flag management |
| [[plugin-github]] | GitHub integration |
| [[plugin-github-runner]] | Self-hosted GitHub Actions runner |
| [[plugin-invitations]] | User invitation system |
| [[plugin-jobs]] | Background job queues |
| [[plugin-link-preview]] | URL preview and metadata extraction |
| [[plugin-mdns]] | mDNS/Bonjour service discovery |
| [[plugin-notifications]] | Notification management |
| [[plugin-search]] | MeiliSearch full-text search |
| [[plugin-subtitle-manager]] | Subtitle download and management |
| [[plugin-tokens]] | Token management |
| [[plugin-torrent-manager]] | Torrent client integration |
| [[plugin-vpn]] | VPN/WireGuard management |
| [[plugin-webhooks]] | Webhook delivery and management |

</details>

<details>
<summary><strong>Pro Plugins (52)</strong> — License key required, starting at $0.99/mo</summary>

**AI and Automation:** [[plugin-ai]] · [[plugin-claw]] · [[plugin-claw-web]] · [[plugin-mux]] · [[plugin-voice]] · [[plugin-browser]] · [[plugin-bots]] · [[plugin-workflows]]

**Communication:** [[plugin-chat]] · [[plugin-livekit]] · [[plugin-streaming]] · [[plugin-realtime]] · [[plugin-podcast]] · [[plugin-meetings]] · [[plugin-recording]]

**Commerce and Payments:** [[plugin-commerce]] · [[plugin-stripe]] · [[plugin-paypal]] · [[plugin-shopify]] · [[plugin-subscription]] · [[plugin-checkout]] · [[plugin-donorbox-pro]] · [[plugin-entitlements]]

**Content and Social:** [[plugin-cms]] · [[plugin-blog]] · [[plugin-pages]] · [[plugin-social]] · [[plugin-activity-feed]] · [[plugin-moderation]] · [[plugin-knowledge-base]] · [[plugin-support]] · [[plugin-documents]] · [[plugin-calendar]] · [[plugin-post]]

**Email Providers:** [[plugin-sendgrid]] · [[plugin-mailgun]] · [[plugin-postmark]] · [[plugin-twilio]]

**Security and Auth:** [[plugin-auth]] · [[plugin-sso]] · [[plugin-saml]] · [[plugin-ldap]] · [[plugin-oauth-providers]] · [[plugin-access-controls]] · [[plugin-idme]] · [[plugin-compliance]] · [[plugin-admin-api]] · [[plugin-audit]] · [[plugin-waf]] · [[plugin-rate-limit]]

**Infrastructure:** [[plugin-google]] · [[plugin-cloudflare]] · [[plugin-object-storage]] · [[plugin-cdn]] · [[plugin-observability]] · [[plugin-ddns]] · [[plugin-devices]] · [[plugin-vpn]] · [[plugin-mdns]]

**Media:** [[plugin-media]] · [[plugin-media-processing]] · [[plugin-file-processing]] · [[plugin-photos]] · [[plugin-stream-gateway]] · [[plugin-epg]] · [[plugin-transcoder]] · [[plugin-thumb]] · [[plugin-watermark]] · [[plugin-drm]]

**Productivity:** [[plugin-cron-pro]] · [[plugin-notify-pro]] · [[plugin-scheduler]] · [[plugin-backup-pro]] · [[plugin-analytics]] · [[plugin-reports]] · [[plugin-export]] · [[plugin-import]]

**Specialized:** [[plugin-tmdb]] · [[plugin-game-metadata]] · [[plugin-retro-gaming]] · [[plugin-rom-discovery]] · [[plugin-sports]] · [[plugin-geocoding]] · [[plugin-geolocation]] · [[plugin-home]] · [[plugin-web3]] · [[plugin-flow]] · [[plugin-content-acquisition]] · [[plugin-content-progress]] · [[plugin-torrent-manager]] · [[plugin-subtitle-manager]]

</details>

---

## Guides

Step-by-step guides for common workflows.

| Guide | Description |
|-------|-------------|
| [[Guide-Production-Deployment]] | Take your local stack to a production server |
| [[Guide-SSL-Setup]] | SSL certificates: auto-generated local, Let's Encrypt, and wildcard |
| [[Guide-Multi-Tenancy]] | Enterprise multi-tenant setup with row-level isolation |
| [[Guide-Security-Hardening]] | Lock down your production deployment |
| [[Guide-Monitoring-Setup]] | Set up Grafana dashboards, alerting, and log aggregation |
| [[Guide-Backup-Restore]] | Automated backups, cloud storage, and disaster recovery |
| [[Guide-Custom-Services]] | Add your own microservices using 40+ templates |
| [[Guide-Migration-from-v1]] | Migrate from the legacy Bash CLI |

---

## Architecture

| Page | Covers |
|------|--------|
| [[Architecture]] | System overview, core services, data flow, service isolation |
| [[Service-Graph]] | Visual service dependency graph |
| [[Compose-Generation]] | How `nself build` generates docker-compose.yml |
| [[Nginx-Generation]] | How nginx.conf is generated from project config |
| [[Config-System]] | System-level architecture and Docker networking |

---

## Security

| Page | Covers |
|------|--------|
| [[Security-Policy]] | Responsible disclosure and reporting |
| [[Security-Architecture]] | How nSelf isolates and hardens services |
| [[Security-Hardening]] | Production hardening checklist |

---

## Brand

| Page | Covers |
|------|--------|
| [[Branding]] | Brand guidelines and usage |
| [[Brand-Assets]] | Downloadable logos, icons, and assets |
| [[Logo-Usage]] | Logo usage rules and clear space |
| [[Color-Palette]] | Official colors and usage |

---

## Contributing

| Page | Covers |
|------|--------|
| [[Contributing]] | How to contribute to nSelf |
| [[Dev-Setup]] | Set up a local development environment |
| [[Plugin-Dev-Guide]] | Build and publish your own plugin |
| [[Release-Process]] | How releases are built and published |

---

## Resources

| Resource | Link |
|----------|------|
| [[Changelog]] | Version history and release notes |
| [[FAQ]] | Common questions and answers |
| [GitHub Repository](https://github.com/nself-org/cli) | Source code |
| [Report Issues](https://github.com/nself-org/cli/issues) | Bug reports and feature requests |
| [Discussions](https://github.com/nself-org/cli/discussions) | Community conversations |
| [nself.org](https://nself.org) | Official website |
| [docs.nself.org](https://docs.nself.org) | Online documentation |
