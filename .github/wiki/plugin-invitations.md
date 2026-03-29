# Invitations Plugin

> User invitation system with email/SMS delivery and token-based join links. **Free — MIT licensed.**

## Install

```bash
nself plugin install invitations
```

## What It Does

Manages user invitations for your application. Generate token-based invite links, deliver them via email or SMS, track acceptance status, and enforce expiry. Integrates with nSelf Auth so accepted invitations automatically provision user accounts.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `INVITATIONS_PORT` | `3402` | Invitations service port |
| `INVITATIONS_EXPIRY_HOURS` | `72` | Hours before invite expires |
| `INVITATIONS_BASE_URL` | — | Your app URL for invite links |
| `INVITATIONS_SMTP_HOST` | — | SMTP host for email delivery |

## Ports

| Port | Purpose |
|------|---------|
| 3402 | Invitations REST API |

## Database Tables

1 table added to your Postgres database:
- `np_invitations_invitations` — invitation records, tokens, and status

## Nginx Routes

None — invitations service is internal only.

## API

```
GET  /health                    — Health check
POST /invitations               — Create and send invitation
GET  /invitations               — List invitations
GET  /invitations/{token}       — Verify an invite token
POST /invitations/{token}/accept — Accept an invitation
DELETE /invitations/{id}        — Revoke an invitation
```
