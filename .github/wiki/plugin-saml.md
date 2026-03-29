# SAML Plugin

> SAML 2.0 identity provider and service provider integration. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install saml
```

## What It Does

Implements full SAML 2.0 support — both as a Service Provider (SP) that accepts assertions from an external IdP, and optionally as an Identity Provider (IdP) for your own services. Handles SAML metadata exchange, assertion validation, attribute mapping, and session management. Works standalone or as part of the `sso` plugin.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SAML_PORT` | `3055` | SAML service port |
| `SAML_SP_ENTITY_ID` | — | SP Entity ID (your app's SAML identifier) |
| `SAML_IDP_METADATA_URL` | — | IdP metadata XML URL |
| `SAML_CERT_FILE` | — | Path to SP certificate file |
| `SAML_KEY_FILE` | — | Path to SP private key file |
| `SAML_ATTRIBUTE_EMAIL` | `email` | SAML attribute for user email |
| `SAML_ATTRIBUTE_ROLES` | `groups` | SAML attribute for role mapping |

## Ports

| Port | Purpose |
|------|---------|
| 3055 | SAML service REST API |

## Database Tables

3 tables added to your Postgres database:
- `np_saml_providers` — configured IdP providers
- `np_saml_sessions` — active SAML sessions
- `np_saml_audit` — assertion validation log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/saml/acs` | Assertion Consumer Service (ACS) endpoint |
| `/saml/metadata` | SP metadata XML |
| `/saml/slo` | Single Logout (SLO) endpoint |
