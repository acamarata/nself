# Feature: nCloud

nCloud is the managed hosting service for nSelf. Instead of setting up your own server, you pay a monthly fee and get a dedicated Hetzner VPS with the full nSelf stack pre-installed, monitored, and maintained.

**Status:** Pricing page and provisioning flow implemented. Full managed service planned.
**Console:** cloud.nself.org
**Architecture doc:** `.claude/docs/ncloud-architecture.md`

---

## How It Works

1. Sign up at cloud.nself.org.
2. Pick a server tier.
3. Pay via Stripe.
4. A dedicated Hetzner VPS is provisioned automatically (under 3 minutes).
5. You receive root SSH access, a Hasura console URL, a GraphQL endpoint, and an admin panel.
6. nSelf monitors the server. You build your app.

Every nCloud customer gets a dedicated server. There is no shared hosting, no noisy neighbors, no resource contention. Your data stays on your server.

---

## Pricing

Pricing follows a transparent model: Hetzner's cost plus a $2/month management fee. All servers are in Falkenstein, Germany (fsn1).

| Tier | Hetzner Type | vCPU | RAM | Disk | Hetzner Cost | nCloud Price |
|------|-------------|------|-----|------|-------------|-------------|
| **Starter** | CX23 | 2 | 4 GB | 40 GB | ~$4/mo | **~$6/mo** |
| **Standard** | CX33 | 4 | 8 GB | 80 GB | ~$7/mo | **~$9/mo** |
| **Performance** | CX43 | 8 | 16 GB | 160 GB | ~$14/mo | **~$16/mo** |
| **ARM Starter** | CAX11 | 2 | 4 GB | 40 GB | ~$4.50/mo | **~$6.50/mo** |

Upgrades and downgrades happen any time via the Hetzner API. Resize takes under 5 minutes.

Plugin licenses are billed separately through the existing nSelf licensing system.

---

## What's Included

Every nCloud server comes with:

- **Full nSelf stack.** PostgreSQL, Hasura GraphQL, Auth, Nginx with SSL.
- **Your plugin tier.** Plugins matching your license key are pre-installed.
- **Server hardening.** fail2ban, UFW, SSH key-only access, security headers.
- **Monitoring.** Prometheus, Grafana, Loki, and the full monitoring bundle.
- **Automatic SSL.** Let's Encrypt certificates configured on provisioning.
- **Root SSH access.** Full control over your server.
- **DNS subdomain.** `{username}.ncloud.nself.org` configured automatically.

---

## Cloud Console (cloud.nself.org)

The web console is a Next.js app in the web/ monorepo (`web/cloud`). It provides:

| Feature | Description |
|---------|------------|
| **Dashboard** | Server status, resource usage (CPU, RAM, disk), uptime |
| **Server management** | Start, stop, restart, resize, rebuild |
| **Plugin management** | Install and remove plugins (delegates to nSelf CLI via SSH) |
| **Logs** | Real-time log streaming from your server |
| **Backups** | Schedule and restore from Hetzner snapshots |
| **DNS** | Add custom domains, auto-configure SSL |
| **Billing** | Current plan, usage history, invoices, upgrade/downgrade |
| **License** | Key display (masked), tier badge, Stripe customer portal |

All management actions execute via SSH to your server. The console queues actions through a job system that connects to your VPS and runs nSelf CLI commands. No management agent runs on your server.

---

## Provisioning

When a new server is ordered, a cloud-init script runs on first boot:

1. System updates and hardening (fail2ban, UFW, SSH key-only)
2. Install nSelf CLI (`curl -fsSL install.nself.org | bash`)
3. `nself init --full` with pre-configured environment
4. Install plugins matching the user's license tier
5. `nself build && nself start`
6. Configure Nginx with SSL (Let's Encrypt)
7. Report ready status to ping_api

Target: under 3 minutes from payment confirmation to a running stack.

---

## Server Lifecycle

```
ncloud_servers table:
  user_id        → auth user
  hetzner_id     → Hetzner server ID
  ipv4           → server IP
  tier           → current Hetzner server type
  status         → provisioning | active | suspended | deleted
  plugins        → installed plugin list
  custom_domains → configured domains
```

- **Failed payment:** 3-day grace period, then server suspended (not deleted).
- **Cancellation:** 7-day grace period with data export option, then server deleted.
- **Disaster recovery:** Server rebuild via `POST /servers/{id}/actions/rebuild`.

---

## Monitoring Relay

Each nCloud server runs the nSelf monitoring bundle. A lightweight relay pushes key metrics to the central nSelf monitoring system:

- Server health (CPU, RAM, disk, network)
- Docker container status for all services
- SSL certificate expiry warnings
- Disk usage alerts

Alerts route to the nSelf ops channel and to the customer's configured notification channels (email, Telegram, webhook).

---

## Custom Domains

Users can map their own domains to their nCloud server:

1. Add domain in the cloud console.
2. Follow DNS instructions (A record pointing to server IP).
3. nSelf verifies DNS propagation.
4. The CLI on the server runs `nself domain add {domain}` to configure Nginx and Let's Encrypt.

The default `{username}.ncloud.nself.org` subdomain is always available, managed via Cloudflare API.

---

## Cloud Light (Planned)

A future lower-cost tier for users who do not need a dedicated server:

| | nCloud | Cloud Light |
|---|--------|------------|
| **Price** | $6-16/mo | $1-2/mo |
| **Server** | Dedicated Hetzner VPS | Shared Docker host |
| **Isolation** | Full server isolation | Per-user schema + Docker namespace |
| **SSH** | Root access | No SSH |
| **Plugins** | All supported | Limited set |
| **Best for** | Production apps | Prototyping, small projects |

Cloud Light uses per-user schema isolation in a shared PostgreSQL instance and subdomain routing to per-user Docker containers.

---

## Security

- Servers are hardened on provisioning: SSH key-only, fail2ban, UFW.
- Root access is the customer's responsibility after provisioning.
- nSelf does not store customer SSH private keys.
- Management SSH uses a deploy key pair (public key injected at provisioning).
- All management actions are logged and auditable.
- Each customer gets a dedicated VPS with no shared resources.

---

## Hetzner API Operations

All server lifecycle operations use the Hetzner Cloud API:

| Operation | API | When |
|-----------|-----|------|
| Create | `POST /servers` | New customer signup |
| Delete | `DELETE /servers/{id}` | Account cancellation (after grace period) |
| Resize | `POST /servers/{id}/actions/change_type` | Tier change |
| Rebuild | `POST /servers/{id}/actions/rebuild` | Disaster recovery |
| Snapshot | `POST /servers/{id}/actions/create_image` | Backup |

A single nCloud Hetzner API token manages all customer servers. All servers live in the nSelf Hetzner project.

---

## Related Pages

- [[Plugin-Licensing]] -- plugin pricing (separate from hosting)
- [[Guide-Production-Deployment]] -- self-hosting alternative to nCloud
- [[Guide-SSL-Setup]] -- SSL for self-hosted deployments
- [[Security-Architecture]] -- security model

---

← [[Features]] | [[_Sidebar]]
