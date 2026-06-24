# ID.me Plugin

> Government-grade identity verification via ID.me OAuth for military, veterans, first responders, government employees, teachers, students, and nurses. **Pro plugin.**

> **Requires:** Pro license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install idme
```

## What It Does

The idme plugin integrates with the ID.me OAuth 2.0 platform to verify user identity for seven government-recognized groups. Users complete a verification flow on ID.me and the result is stored in your database. Your application can then gate access, unlock discounts, or surface badges based on verified status. All verification events are written to an audit table for compliance.

## Verification Groups

ID.me verification covers seven distinct groups:

| Group | Who Qualifies |
|-------|--------------|
| Military | Active duty service members |
| Veteran | Honorably discharged veterans |
| First Responder | Police, fire, EMS personnel |
| Government Employee | Federal, state, and local government workers |
| Teacher | K-12 and university educators |
| Student | Enrolled students at accredited institutions |
| Nurse | Licensed nurses and nursing students |

Each group has its own scope in the OAuth flow. Users can verify for multiple groups in a single session or return later to add more.

## OAuth Flow

1. Your app redirects the user to `GET /idme/verifications` with the desired scopes.
2. The user completes ID.me verification in the browser.
3. ID.me calls your registered redirect URI (`IDME_REDIRECT_URI`) with an authorization code.
4. The plugin exchanges the code for tokens, fetches the user attributes, and stores results.
5. Your app queries verification status via `GET /idme/users/{user_id}/verifications`.

Use `IDME_SANDBOX=true` during development to avoid requiring real credentials.

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `IDME_CLIENT_ID` | yes | — | ID.me OAuth application client ID |
| `IDME_CLIENT_SECRET` | yes | — | ID.me OAuth application client secret |
| `IDME_REDIRECT_URI` | yes | — | OAuth redirect URI registered with ID.me |
| `IDME_SCOPES` | no | `openid,email,profile` | OAuth scopes to request |
| `IDME_SANDBOX` | no | `true` | Use ID.me sandbox (`idmelabs.com`) for testing |
| `IDME_WEBHOOK_SECRET` | no | — | Secret to verify ID.me webhook signatures |

Register your application at [developers.id.me](https://developers.id.me) to get `IDME_CLIENT_ID` and `IDME_CLIENT_SECRET`. Set the redirect URI in your ID.me app to match `IDME_REDIRECT_URI`.

## Port

| Port | Purpose |
|------|---------|
| 3820 | ID.me plugin REST API |

## Database Tables

The plugin adds five tables to your Postgres database:

| Table | Purpose |
|-------|---------|
| `np_idme_verifications` | Per-user verification records and status |
| `np_idme_groups` | Verified group memberships |
| `np_idme_badges` | Visual badge assignments per group |
| `np_idme_attributes` | Verified user attributes (branch, rank, affiliation) |
| `np_idme_webhook_events` | Incoming webhook event log |

All tables use `source_account_id` for multi-app isolation.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/idme/` | ID.me verification API |

## Webhooks

ID.me sends real-time events to `/webhooks/idme`. Set `IDME_WEBHOOK_SECRET` and the plugin will verify the signature on each incoming request.

| Event | Description |
|-------|-------------|
| `verification.created` | New verification started |
| `verification.completed` | Verification completed successfully |
| `verification.failed` | Verification failed |
| `group.verified` | User verified for a specific group |
| `group.revoked` | Group verification revoked |

## Docker Image

```bash
docker pull nself/plugin-idme:latest
```

## Related

- [[Plugin-Overview]] - Full plugin catalog
- [[Plugin-Install]] - How to install plugins
- [[Plugin-Licensing]] - License tiers and keys
- [[Plugins-AI-OAuth]] - Other OAuth plugins
- [[Home]]
