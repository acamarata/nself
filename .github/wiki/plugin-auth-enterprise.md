# Auth Enterprise Plugin (Pro — Max Tier)

> MFA enforcement (TOTP + recovery codes) and SSO via SAML 2.0 + OIDC for Google Workspace, Okta, and Microsoft Entra ID. **Pro max-tier plugin.**

> **Requires:** ɳSelf+ license. `nself license set nself_plus_...`

## Install

```bash
nself license set nself_plus_xxxxx...
nself plugin install auth-enterprise
```

## What It Does

Adds enterprise authentication controls to your nSelf deployment. MFA (TOTP + recovery codes) is always active for all users at no cost — it cannot be paywalled per the Security-Always-Free doctrine. SSO (SAML 2.0 + OIDC) requires `NSELF_SSO=true`.

- **TOTP MFA** — RFC 6238, 30-second window, 1-step drift tolerance. Backup via single-use recovery codes (bcrypt-hashed).
- **MFA policy** — per-tenant enforcement: `optional`, `required`, or `role-gated`.
- **SAML 2.0 SSO** — SP-initiated flow with Okta, Google Workspace, Microsoft Entra ID, and any SAML 2.0 IdP.
- **OIDC SSO** — Authorization code flow for Google Workspace and OIDC-compliant providers.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `NSELF_SSO` | `false` | Enable SSO endpoints (`true` to activate) |
| `AUTH_ENTERPRISE_TOTP_ISSUER` | `nSelf` | Name shown in authenticator apps |
| `AUTH_ENTERPRISE_SSO_SP_ENTITY_ID` | — | SAML SP entity ID URI (required for SAML) |
| `AUTH_ENTERPRISE_SSO_ACS_URL` | — | SAML Assertion Consumer Service URL |
| `AUTH_ENTERPRISE_SSO_OIDC_CALLBACK_URL` | — | OIDC redirect URI |

## Ports

| Port | Purpose |
|------|---------|
| 3826 | Auth Enterprise REST API |

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_mfa_enrollments` | TOTP and WebAuthn enrollment records |
| `np_mfa_recovery_codes` | Single-use bcrypt-hashed recovery codes |
| `np_mfa_policies` | Per-tenant MFA enforcement policy |
| `np_sso_providers` | SAML/OIDC IdP configuration |
| `np_sso_sessions` | Active SSO sessions |
| `np_sso_state_cache` | OIDC/SAML state nonce cache |

## Per-Provider Setup

### Google Workspace (OIDC)
1. Create OAuth2 credentials in [Google Cloud Console](https://console.cloud.google.com).
2. Add your `AUTH_ENTERPRISE_SSO_OIDC_CALLBACK_URL` as an authorized redirect URI.
3. Register the provider: `POST /auth/sso/providers` with `"type": "oidc"`, `"issuer_url": "https://accounts.google.com"`.

### Okta (SAML 2.0)
1. Create a SAML app in Okta. Set the ACS URL and SP entity ID to match your env vars.
2. Download the IdP metadata XML from Okta.
3. Register: `POST /auth/sso/providers` with `"type": "saml"` and the metadata XML.

### Microsoft Entra ID (SAML 2.0 or OIDC)
1. Register an enterprise application in Entra ID.
2. For SAML: set the identifier (entity ID) and reply URL (ACS URL) in the SAML settings.
3. For OIDC: copy the client ID + secret and set the redirect URI.

## API

```
GET  /health                          — Liveness check
GET  /auth/mfa/status                 — MFA enrollment status
POST /auth/mfa/totp/setup             — Begin TOTP enrollment
POST /auth/mfa/totp/verify            — Confirm enrollment with live code
POST /auth/mfa/totp/challenge         — Verify TOTP code during login
POST /auth/mfa/recovery               — Consume a recovery code
GET  /auth/mfa/recovery/codes         — List masked recovery codes
POST /auth/mfa/recovery/regenerate    — Replace all recovery codes
GET  /auth/mfa/policy                 — Get tenant MFA policy
PUT  /auth/mfa/policy                 — Set tenant MFA policy (mfa:admin)
GET  /auth/sso/providers              — List SSO providers (SSO-gated)
POST /auth/sso/providers              — Create SSO provider (SSO-gated)
DELETE /auth/sso/providers/{id}       — Remove SSO provider
GET  /auth/sso/providers/{id}/saml/metadata — SP SAML metadata XML
GET  /auth/sso/oidc/{id}/begin        — Start OIDC authorization flow
GET  /auth/sso/oidc/callback          — OIDC authorization code callback
POST /auth/sso/saml/{id}/acs          — SAML ACS endpoint
```

## Security Notes

- MFA is always free and cannot be disabled per the Security-Always-Free doctrine.
- TOTP secrets are stored as plaintext base32; enable pgcrypto for at-rest encryption in production.
- Recovery codes are bcrypt-hashed (cost 10) and never stored in plaintext.
- SSO client secrets should be encrypted by the application before persistence in `np_sso_providers`.
