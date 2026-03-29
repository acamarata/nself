# Auth Plugin (Pro)

> Extended authentication with passkeys, MFA, device codes, and magic links. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install auth
```

## What It Does

Extends nSelf core authentication with advanced flows not included in the base stack. Adds WebAuthn passkey registration and assertion, TOTP-based multi-factor authentication, OAuth device code grants, and magic link email flows. All flows are audited to a persistent log for compliance and investigation.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `AUTH_PLUGIN_PORT` | `3014` | Auth plugin service port |
| `AUTH_PASSKEYS_ENABLED` | `true` | Enable WebAuthn passkey support |
| `AUTH_MFA_ENABLED` | `true` | Enable TOTP multi-factor authentication |
| `AUTH_MAGIC_LINK_EXPIRY` | `15m` | Expiry duration for magic link tokens |

## Ports

| Port | Purpose |
|------|---------|
| 3014 | Auth plugin REST API |

## Database Tables

7 tables added to your Postgres database:
- `np_auth_sessions` — extended session records
- `np_auth_passkeys` — registered WebAuthn passkey credentials
- `np_auth_mfa_configs` — per-user MFA configuration
- `np_auth_device_codes` — OAuth device code grants
- `np_auth_magic_links` — issued magic link tokens
- `np_auth_audit_log` — authentication event audit trail
- `np_auth_trusted_devices` — user-trusted device records

## Nginx Routes

| Route | Target |
|-------|--------|
| `/auth/` | Auth plugin API |
