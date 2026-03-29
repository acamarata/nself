# Guide: SSL / TLS Setup

nSelf manages TLS certificates through Nginx. Three certificate modes are supported.

## Self-Signed Certificates (Default)

nSelf auto-generates a self-signed certificate when you first run `nself build`. This is appropriate for local development and internal networks.

```bash
nself ssl status    # check certificate details and expiry
```

Browsers will show a security warning for self-signed certs — expected behaviour for local dev.

## Custom Certificate Installation

If you have a certificate from a commercial CA (DigiCert, Sectigo, etc.):

1. Place your files on the server:
   ```
   /etc/ssl/certs/nself/cert.pem      # full chain certificate
   /etc/ssl/certs/nself/key.pem       # private key
   ```

2. Set paths in your `.env.prod`:
   ```env
   SSL_CERT_PATH=/etc/ssl/certs/nself/cert.pem
   SSL_KEY_PATH=/etc/ssl/certs/nself/key.pem
   ```

3. Rebuild and restart:
   ```bash
   nself build && nself restart
   ```

## Let's Encrypt (Manual)

Automated ACME certificate renewal is a post-v1.0 feature. For v1.0, use `certbot` manually:

```bash
# Install certbot
apt install certbot

# Stop nginx temporarily
nself stop

# Request certificate
certbot certonly --standalone -d example.com -d '*.example.com'

# Certificates are written to /etc/letsencrypt/live/example.com/
# Copy to nSelf paths:
cp /etc/letsencrypt/live/example.com/fullchain.pem /etc/ssl/certs/nself/cert.pem
cp /etc/letsencrypt/live/example.com/privkey.pem   /etc/ssl/certs/nself/key.pem

# Restart
nself start
```

> **Note:** Automatic ACME renewal (via certbot renew hooks) is planned for a post-v1.0 release.

## SSL Commands

| Command | Description |
|---------|-------------|
| `nself ssl status` | Show certificate path, issuer, and expiry date |
| `nself ssl renew` | Trigger manual certificate renewal prompt |

## See Also

- [[Guide-Production-Deployment]] — full server setup
- [[Guide-Security-Hardening]] — ensure TLS is properly configured
- [[cmd-ssl]] — ssl command reference

---
← [[Home]] | [[_Sidebar]]
