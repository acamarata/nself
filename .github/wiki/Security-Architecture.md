# Security Architecture

nSelf is designed with security as a core concern, not an afterthought. This document describes the security model and key protections built into the CLI and generated infrastructure.

## Container Security

All generated containers include hardened defaults:

```yaml
security_opt:
  - no-new-privileges:true
cap_drop:
  - ALL
```

- **`no-new-privileges`** — prevents privilege escalation via setuid binaries
- **`cap_drop: ALL`** — removes all Linux capabilities from containers by default

Nginx is the only service that retains `NET_BIND_SERVICE` (required for ports 80/443).

## Network Isolation

All internal services bind to `127.0.0.1` only. The Docker bridge network (`{project}_default`) is internal — services communicate via Docker DNS names (e.g., `hasura:8080`).

Only Nginx exposes external ports (80 and 443). All external traffic flows through Nginx, which handles TLS termination and proxying to internal services.

| Service | Binding | External Access |
|---------|---------|----------------|
| PostgreSQL | `127.0.0.1:5432` | Nginx only (not exposed) |
| Hasura | `127.0.0.1:8080` | Via Nginx proxy |
| Auth | `127.0.0.1:4000` | Via Nginx proxy |
| Nginx | `0.0.0.0:80,443` | Direct |

## Secret Management

Secrets (passwords, API keys, JWT secrets) are managed through the `.env` cascade:

- `.env.secrets` — sensitive values, **never committed to git**
- Secrets are passed to containers as environment variables, not files
- `.env.secrets` must be in `.gitignore` (verified by `nself doctor`)
- Secret values are **redacted** from all CLI output (`***REDACTED***`)

Secret env vars matching these patterns are automatically redacted:
`*_PASSWORD`, `*_SECRET`, `*_KEY`, `*_TOKEN`

## Security Headers

nSelf's Nginx configuration includes security headers on all server blocks:

```nginx
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Content-Security-Policy "default-src 'self'" always;
server_tokens off;
```

## Auth Rate Limiting

Auth endpoints are rate-limited to prevent brute-force attacks:

- Default: **30 requests/minute** per IP
- Burst: 5 requests
- Configurable via `AUTH_RATE_LIMIT` env var

## Input Sanitisation

All user inputs are validated via `internal/sanitize` before being used in generated configs:

| Input | Constraint |
|-------|-----------|
| Project name | Lowercase, 2–30 chars, alphanumeric + hyphens |
| Base domain | Valid FQDN |
| Port numbers | 1024–65535, not in reserved list |
| Custom service names | Lowercase, alphanumeric + hyphens/underscores |
| Plugin versions | Anchored semver regex |
| Backup file paths | No path traversal (`../`) |

## Plugin Integrity

Plugin manifests are decoded with `DisallowUnknownFields` to prevent field injection attacks. All fields are validated after decode:
- Version strings must match anchored semver
- Plugin download URLs must use HTTPS from allowed domains
- Checksums are verified (SHA-256) before installation

## File Permissions

nSelf applies least-privilege file permissions:

| File/Dir | Permission |
|----------|-----------|
| `.env`, `.env.*` | `0600` (owner read/write only) |
| Backup directories | `0700` |
| SSL certificate directory | `0750` |
| Plugin cache files | `0600` |

## TLS

The nSelf CLI itself enforces TLS 1.2 minimum for all outbound HTTPS connections (registry, license validation, update checks). Weaker protocol versions are rejected.

## Deferred to v1.1

The following security improvements are planned for a future release:

- Non-root container users for Postgres, Auth, Functions
- Docker socket proxy for Admin service
- Binary signature verification (GPG)
- Automatic ACME certificate renewal

## See Also

- [[Security-Policy]] — how to report vulnerabilities
- [[Security-Hardening]] — production checklist
- [[Guide-Security-Hardening]] — step-by-step hardening guide

---
← [[Home]] | [[_Sidebar]]
