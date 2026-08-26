# Guide: Production Deployment

This guide covers deploying ɳSelf to a production VPS from scratch.

## Server Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 2 vCPU | 4 vCPU |
| RAM | 4 GB | 8 GB |
| Disk | 40 GB SSD | 80 GB SSD |
| OS | Ubuntu 22.04 long-term support | Ubuntu 22.04 long-term support |

Tested providers: Hetzner CX23 (2 vCPU / 4 GB) meets the minimum for a standard stack.

## Firewall Rules

Open only the ports your server needs:

| Port | Direction | Purpose |
|------|-----------|---------|
| 80 | Inbound | HTTP (redirects to HTTPS) |
| 443 | Inbound | HTTPS |
| 22 | Inbound | SSH admin access |

**Close to external access:** 5432 (Postgres), 8080 (Hasura), 4000 (Auth), and all other internal service ports. ɳSelf routes all external traffic through Nginx on 80/443.

Example with `ufw`:

```bash
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw default deny incoming
ufw enable
```

## DNS Setup

Create an A record pointing your domain to the server IP:

```
A  example.com          → 1.2.3.4
A  *.example.com        → 1.2.3.4   # wildcard for subdomains
```

Wait for DNS propagation (up to 24 hours) before running TLS setup.

## Installation

```bash
# Install nSelf CLI
/bin/bash -c "$(curl -fsSL https://install.nself.org)"

# Verify
nself version
```

## Initialise and Build

```bash
# Create project with all environment files, including .env.prod
nself init myproject --full

# Review and edit .env.prod
# Set: BASE_DOMAIN, HASURA_GRAPHQL_ADMIN_SECRET, JWT secrets, Postgres password

# Generate docker-compose and nginx config for the production environment
ENV=prod nself build

# Start all services
ENV=prod nself start
```

## Verify Deployment

```bash
# Check all services are healthy
nself health

# List service URLs
nself urls
```

## Ongoing Maintenance

| Task | Command |
|------|---------|
| Update CLI | `nself update` |
| Apply config changes | `nself build && nself restart` |
| View service status | `nself status` |
| Tail logs | `nself logs` |
| Check health | `nself health` |

## See Also

- [[Guide-SSL-Setup]], TLS certificate configuration
- [[Guide-Security-Hardening]], post-deployment security checklist
- [[Guide-Monitoring-Setup]], enable metrics and alerting
- [[Guide-Backup-Restore]], set up automated backups

---
← [[Home]] | [[_Sidebar]]
