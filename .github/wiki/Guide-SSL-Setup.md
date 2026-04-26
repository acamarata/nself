# Guide: SSL / TLS Setup

ɳSelf manages TLS certificates through nginx. Three certificate modes are supported.

## Self-Signed Certificates (Default)

ɳSelf auto-generates a self-signed certificate when you first run `nself build`. This is appropriate for local development and internal networks.

```bash
nself ssl status    # check certificate details and expiry
```

Browsers will show a security warning for self-signed certs, expected behaviour for local dev.

## Base Domain Certificates (DNS-01)

Use `nself ssl setup` to provision a wildcard or multi-domain Let's Encrypt certificate for your configured `BASE_DOMAIN` via DNS-01 challenge. This requires your DNS provider credentials to be in place before running.

```bash
# Wildcard certificate via Cloudflare
nself ssl setup --provider cloudflare --wildcard --email admin@example.com

# Per-subdomain certificate via Route53
nself ssl setup --provider route53 --email admin@example.com

# Enable automatic renewal via systemd timer (Linux)
nself ssl setup --provider cloudflare --wildcard --install-cron
```

Supported providers: `cloudflare`, `route53`, `digitalocean`. For other CAs, place credentials in `/etc/letsencrypt/{provider}.ini` and use `--provider custom`.

Certificates land in `ssl/{domain}/` inside the project directory. nginx reads them at `/etc/nginx/ssl/{domain}/` via the `./ssl:/etc/nginx/ssl:ro` mount.

## Custom Domain Certificates (HTTP-01)

Use `nself ssl add` to provision a certificate for an external custom domain (e.g., a white-labelled subdomain or a partner's domain). This uses HTTP-01 challenge, no DNS provider configuration needed.

```bash
# Add certificate and proxy traffic to a backend container
nself ssl add portal.example.com --upstream portal-app:8080

# Add certificate without configuring an upstream yet
nself ssl add portal.example.com
```

After certbot completes, the command:

1. Writes `nginx/conf.d/custom-{domain}.conf` with a full HTTPS server block (HTTP/2, security headers, HTTP-to-HTTPS redirect).
2. Tests the nginx config (`nginx -t`).
3. Reloads nginx.

The generated conf uses `proxy_pass http://{upstream}` when `--upstream` is provided, or returns a `200` response until an upstream is configured.

To update the upstream later, edit `nginx/conf.d/custom-{domain}.conf` and reload:

```bash
docker compose exec nginx nginx -s reload
```

## Custom Certificate Installation

If you have a certificate from a commercial CA (DigiCert, Sectigo, etc.):

1. Place your files in the project's `ssl/{domain}/` directory:
   ```
   ssl/{domain}/fullchain.pem
   ssl/{domain}/privkey.pem
   ```

2. Write an nginx server block to `nginx/conf.d/custom-{domain}.conf` following the same pattern that `nself ssl add` generates.

3. Test and reload nginx:
   ```bash
   docker compose exec nginx nginx -t
   docker compose exec nginx nginx -s reload
   ```

## SSL Commands

| Command | Description |
|---------|-------------|
| `nself ssl status` | Show certificate path, issuer, and expiry date |
| `nself ssl renew` | Trigger manual certificate renewal prompt |
| `nself ssl setup` | Provision a wildcard/multi-domain cert via DNS-01 |
| `nself ssl add <domain>` | Provision a cert for a single custom domain via HTTP-01 |

## See Also

- [[cmd-ssl]], ssl command reference
- [[Guide-Production-Deployment]], full server setup
- [[Guide-Security-Hardening]], ensure TLS is properly configured

---
← [[Home]] | [[_Sidebar]]
