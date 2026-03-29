> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# LDAP Plugin

> LDAP and Active Directory sync — map AD groups to nSelf roles. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install ldap
```

## What It Does

Syncs users and groups from LDAP or Microsoft Active Directory into your nSelf user database. Maps AD security groups to nSelf roles for automatic permission management. Supports both periodic sync and on-login bind for authentication. Works alongside nSelf Auth for hybrid local + directory authentication.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `LDAP_PORT` | `3056` | LDAP plugin service port |
| `LDAP_SERVER` | — | LDAP server hostname |
| `LDAP_SERVER_PORT` | `389` | LDAP server port (636 for LDAPS) |
| `LDAP_USE_TLS` | `true` | Use TLS/LDAPS |
| `LDAP_BIND_DN` | — | Service account DN for bind |
| `LDAP_BIND_PASSWORD` | — | Service account password |
| `LDAP_USER_BASE_DN` | — | Base DN for user search |
| `LDAP_GROUP_BASE_DN` | — | Base DN for group search |
| `LDAP_SYNC_INTERVAL` | `3600` | Sync interval in seconds |

## Ports

| Port | Purpose |
|------|---------|
| 3056 | LDAP sync service REST API |

## Database Tables

4 tables added to your Postgres database:
- `np_ldap_users` — synced LDAP user records
- `np_ldap_groups` — synced LDAP groups
- `np_ldap_group_mappings` — group-to-role mappings
- `np_ldap_sync_log` — sync history

## Nginx Routes

None — LDAP sync is an internal background service.
