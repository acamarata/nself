# Server Setup

Practical server configuration for nself deployments. These steps apply to any
Linux VPS (Hetzner CX*, DigitalOcean Droplet, AWS EC2, etc.).

## Swap Configuration

Plugin-heavy installs can exhaust RAM during startup — especially when ai, mux, or
claw are running alongside the core stack. Swap provides a safety buffer.

Recommended swap sizes:

| Server RAM | Plugins installed | Recommended swap |
| --- | --- | --- |
| 2 GB (CX11/CX21) | 0–3 | 2 GB |
| 4 GB (CX22) | 3–6 | 2 GB |
| 8 GB (CX32) | 6+ or ai/mux | 4 GB |
| 16 GB+ | any | 4 GB |

```bash
# Create a 4 GB swapfile (adjust size as needed)
fallocate -l 4G /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile

# Make swap permanent across reboots
echo '/swapfile none swap sw 0 0' >> /etc/fstab
```

Verify swap is active:

```bash
free -h
swapon --show
```

## Cloudflare Free SSL Limitation

Cloudflare's free SSL covers one level of wildcard: `*.example.com`. It does NOT
cover second-level subdomains like `webhook.ai.example.com`.

This matters for webhook endpoints — Telegram bots, Google Pub/Sub push, and similar
services require a valid TLS cert on the webhook URL. If you proxy through Cloudflare,
use a flat subdomain instead of a nested one:

```
# Works with Cloudflare free SSL:
webhooks.example.com/telegram
webhooks.example.com/pubsub

# Does NOT work with Cloudflare free SSL:
webhook.ai.example.com
callback.notify.example.com
```

If you need second-level subdomains with valid TLS, either:
- Upgrade to Cloudflare Pro (covers `*.*.example.com`), or
- Use Let's Encrypt directly via nself's built-in cert management (`nself ssl`).

## UFW Firewall (Recommended)

```bash
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP (redirects to HTTPS via nginx)
ufw allow 443/tcp   # HTTPS
ufw enable
```

Docker bypasses UFW by default. See [Docker and UFW](https://docs.docker.com/network/iptables/)
for how to prevent Docker from opening ports directly.

## Time Sync

nself services (JWT expiry, cron scheduling, log timestamps) depend on correct system time.
Most cloud VMs have time sync enabled by default. Verify:

```bash
timedatectl status
```

If `NTP service: inactive`, enable it:

```bash
timedatectl set-ntp true
```
