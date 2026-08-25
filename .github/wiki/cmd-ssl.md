# nself ssl

<!-- BEGIN PROSE:summary -->
> Manage SSL certificates for ɳSelf services and custom domains.
<!-- END PROSE:summary -->

## Synopsis

```
nself ssl <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself ssl` manages the SSL certificates used by nginx to serve HTTPS traffic for all ɳSelf services. Certificates are generated automatically during `nself build`, but you can use this command to check their status or force regeneration without a full rebuild.

Use `nself ssl setup` to provision wildcard certificates via DNS-01 challenge for the configured base domain. Use `nself ssl add` to provision a certificate for a single external custom domain and generate the corresponding nginx server block automatically.

Certificates written by `ssl setup` and `ssl add` land in `ssl/{domain}/` inside the project directory, which nginx reads via its `./ssl:/etc/nginx/ssl:ro` volume mount.

## nself ssl setup

Provisions SSL certificates using certbot with DNS-01 validation. Supports wildcard certificates for `*.domain`.

```
nself ssl setup [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--provider` | `cloudflare` | DNS provider (`cloudflare`, `route53`, `digitalocean`, `custom`) |
| `--wildcard` | `false` | Request a wildcard certificate (`*.domain`) |
| `--email` | (from `ADMIN_EMAIL`) | Email address for Let's Encrypt registration |
| `--staging` | `false` | Use the Let's Encrypt staging environment |
| `--install-cron` | `false` | Install a systemd timer for automatic renewal (Linux only) |

### Examples

```bash
# Wildcard certificate via Cloudflare DNS
nself ssl setup --provider cloudflare --wildcard

# Single-domain certificate via Route53
nself ssl setup --provider route53 --domain api.example.com

# Staging run (does not consume rate-limit quota)
nself ssl setup --provider cloudflare --staging
```

## nself ssl add

Provisions an SSL certificate for a single custom domain via HTTP-01 challenge (no DNS provider needed). After certbot succeeds, writes an nginx server block to `nginx/conf.d/custom-{domain}.conf` and reloads nginx.

Certificates are stored in `ssl/{domain}/` so nginx can read them at `/etc/nginx/ssl/{domain}/` inside the container.

```
nself ssl add <domain> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--upstream` | (none) | Backend service to proxy to (`host:port`). When omitted, a 200 placeholder response is returned until an upstream is configured. |

### Examples

```bash
# Add certificate with proxy to an app container on port 3000
nself ssl add custom.example.com --upstream app:3000

# Add certificate without upstream (returns 200 placeholder)
nself ssl add custom.example.com
```

The generated conf file (`nginx/conf.d/custom-custom-example-com.conf`) includes:

- HTTP-to-HTTPS redirect on port 80
- TLS on port 443 with HTTP/2
- Security headers: `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Strict-Transport-Security`
- `proxy_pass` block (when `--upstream` is set) or placeholder `return 200`
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `add` | Provision an SSL certificate for a single domain |
| `renew` | Reload nginx and optionally renew certificates |
| `setup` | Set up SSL certificates via DNS-01 challenge |
| `status` | Show SSL certificate status |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Check certificate status and expiry
nself ssl status

# Force certificate regeneration
nself ssl renew

# Provision wildcard via Cloudflare
nself ssl setup --provider cloudflare --wildcard --email admin@example.com

# Add custom domain with backend proxy
nself ssl add portal.example.com --upstream portal-app:8080
```

**Sample `status` output:**

```
Certificate: ssl/cert.pem
  Issued to:  *.localhost, localhost
  Expires:    2027-03-28 (730 days remaining)
  CA trust:   trusted (mkcert CA installed)
  SANs:       localhost, *.localhost, api.localhost, auth.localhost
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Guide-SSL-Setup]], full SSL setup walkthrough
- [[Config-Nginx]], nginx configuration reference
- [[Guide-Production-Deployment]], production server setup
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
