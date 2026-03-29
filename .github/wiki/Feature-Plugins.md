# Feature: Plugins

The nSelf plugin system extends your stack with additional services, API routes, and capabilities — without writing infrastructure code.

## How Plugins Work

A plugin is a manifest + Docker Compose overlay + optional Nginx config. When you install and build:

1. `nself plugin install <name>` — downloads and caches the plugin
2. `nself build` — merges the plugin's compose overlay into `docker-compose.yml` and injects Nginx routes
3. `nself start` / `nself restart` — runs the updated stack

## Plugin Tiers

| Tier | Price | Included |
|------|-------|---------|
| Free | $0 | 25 free plugins — no license key required |
| Basic | $0.99/mo | All 59 pro plugins |
| Pro | $1.99/mo | Basic + AI suite |
| Elite | $4.99/mo | Pro + email support |
| Business | $9.99/mo | Elite + 24h support |

See [[Plugin-Licensing]] for full tier details.

## Free Plugins (25)

monitoring, mlflow, backup, cron, notify, jobs, feature-flags, webhooks, notifications, search, invitations, link-preview, github, github-runner, content-acquisition, content-progress, torrent-manager, subtitle-manager, tokens, vpn, mdns, donorbox (+ 3 pending naming)

## Pro Plugins (59)

Categories: AI & Automation, Communication, Media & Processing, Content & Social, Commerce & Payments, Auth & Security, Infrastructure & Integration, Productivity, Gaming & Specialised.

See [[Plugin-Overview]] for the complete list.

## Plugin Commands

```bash
nself plugin install <name>       # install a plugin
nself plugin remove <name>        # remove a plugin
nself plugin update <name>        # update to latest version
nself plugin list                 # list installed plugins
nself plugin registry             # browse available plugins
```

## Building Your Own Plugin

See [[Plugin-Dev-Guide]] to build and publish a custom plugin.

## See Also

- [[Plugin-Overview]] — browse all plugins
- [[Plugin-Architecture]] — technical details
- [[Plugin-Licensing]] — tier and pricing details
- [[Plugin-Dev-Guide]] — build a plugin

---
← [[Home]] | [[_Sidebar]]
