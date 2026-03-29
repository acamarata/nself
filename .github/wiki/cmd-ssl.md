# nself ssl

> Manage SSL certificates for nSelf services.

## Synopsis

```
nself ssl <subcommand>
```

## Description

`nself ssl` manages the SSL certificates used by nginx to serve HTTPS traffic for all nSelf services. Certificates are generated automatically during `nself build`, but you can use this command to check their status or force regeneration without a full rebuild.

Certificate generation uses `mkcert` when available (produces browser-trusted certificates for local development) or falls back to OpenSSL self-signed certificates with Subject Alternative Names (SANs) covering all configured subdomains. Certificates are stored in the project's `ssl/` directory.

Use `nself ssl status` to verify expiry dates before a deployment or after changing your `BASE_DOMAIN`.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `status` | Show certificate expiry, covered domains (SANs), and CA trust status |
| `renew` | Force regeneration of SSL certificates |

## Examples

```bash
# Check certificate status and expiry
nself ssl status

# Force certificate regeneration
nself ssl renew
```

**Sample `status` output:**

```
Certificate: ssl/cert.pem
  Issued to:  *.localhost, localhost
  Expires:    2027-03-28 (730 days remaining)
  CA trust:   ✓ trusted (mkcert CA installed)
  SANs:       localhost, *.localhost, api.localhost, auth.localhost
```

← [[Commands]] | [[Home]] →
