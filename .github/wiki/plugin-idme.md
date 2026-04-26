# ID.me Plugin

> Government-grade identity verification via ID.me for age, military, student, and first responder checks. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install idme
```

## What It Does

Integrates with the ID.me identity verification platform to confirm user attributes such as age, military service, student enrollment, and first responder status. Verification results gate access controls or enable discounts within your application. All verification events are recorded to an audit log for compliance.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `IDME_PORT` | `3010` | ID.me plugin service port |
| `IDME_CLIENT_ID` | — | ID.me OAuth application client ID |
| `IDME_CLIENT_SECRET` | — | ID.me OAuth application client secret |
| `IDME_REDIRECT_URI` | — | OAuth redirect URI registered with ID.me |
| `IDME_SANDBOX` | `true` | Use ID.me sandbox environment for testing |

## Ports

| Port | Purpose |
|------|---------|
| 3010 | ID.me plugin REST API |

## Database Tables

5 tables added to your Postgres database:
- `np_idme_verifications`, user verification records
- `np_idme_tokens`, OAuth token storage
- `np_idme_groups`, ID.me group/community memberships
- `np_idme_affiliations`, verified user affiliations
- `np_idme_audit_log`, verification event audit trail

## Nginx Routes

| Route | Target |
|-------|--------|
| `/idme/` | ID.me verification API |
