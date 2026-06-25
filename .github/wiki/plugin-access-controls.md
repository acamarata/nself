# Access Controls Plugin

> RBAC + ABAC authorization system with role hierarchy, pattern matching, and permission caching. **Pro plugin.**

> **Requires:** a Pro license tier or higher. Set it with `nself license set nself_pro_...` before installing.

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install access-controls
```

## What It Does

Extends ɳSelf Auth with fine-grained authorization. It implements both Role-Based Access Control (RBAC) and Attribute-Based Access Control (ABAC) in one policy engine on port 3027.

- **RBAC mode:** define roles with fine-grained permissions. Roles support a parent-child hierarchy, so a child role inherits its parent's permissions. Permissions accept wildcard patterns such as `posts:*` or `*`.
- **ABAC mode:** define policy rules with conditions and patterns that evaluate at request time. Policies carry a priority and an enabled flag, and the engine resolves them against the request context.
- **Scoped roles:** assign a role with a specific scope, for example a channel moderator.
- **Caching:** permission results are cached in memory with a configurable TTL for fast repeat checks.
- **Multi-app isolation:** every record is scoped by `source_account_id`, so independent apps in one nSelf deploy never see each other's roles or policies.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ACL_PLUGIN_PORT` | `3027` | Access controls service port |
| `ACL_CACHE_TTL_SECONDS` | `300` | Permission cache TTL in seconds |
| `ACL_MAX_ROLE_DEPTH` | `10` | Maximum depth for role-hierarchy inheritance |
| `ACL_DEFAULT_DENY` | `true` | Deny when no role or policy grants access |
| `ACL_API_KEY` | (unset) | API key for service-to-service auth |
| `ACCESS_CONTROLS_ALLOWED_ORIGINS` | (unset) | Comma-separated CORS allow-list |
| `DATABASE_URL` | (required) | Postgres connection string |

## Ports

| Port | Purpose |
|------|---------|
| 3027 | Access controls REST API |

## Database Tables

6 tables are added to your Postgres database. Every table is scoped by `source_account_id` for multi-app isolation:

- `acl_roles`, role definitions with parent-child hierarchy
- `acl_permissions`, permission definitions
- `acl_role_permissions`, role-to-permission mappings
- `acl_user_roles`, user-to-role assignments
- `acl_policies`, ABAC policy rules with priority and enabled flag
- `acl_webhook_events`, event log

## API Endpoints

The service exposes a versioned REST API under `/v1`:

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/v1/authorize` | Check a single authorization decision |
| POST | `/v1/authorize/batch` | Check authorization in batch |
| POST | `/v1/roles` | Create a role |
| GET | `/v1/roles` | List roles |
| GET | `/v1/roles/hierarchy` | Get the role hierarchy |
| POST | `/v1/permissions` | Create a permission |
| POST | `/v1/users/:userId/roles` | Assign a role to a user |
| GET | `/v1/users/:userId/permissions` | Get a user's effective permissions |
| POST | `/v1/policies` | Create an ABAC policy |
| POST | `/v1/cache/invalidate` | Invalidate the permission cache |
| GET | `/v1/cache/stats` | Read cache statistics |
| GET | `/health` | Health check |

## Hasura Integration

Use the `/v1/authorize` endpoint from your app or a Hasura action to gate access before a mutation or query runs. Because the engine owns the policy decision, you do not duplicate the rules in Hasura permissions. Hasura still enforces per-row `source_account_id` filtering, and this plugin enforces the role and policy decision on top.

---

[[Home]] · [[Plugin-Overview]]
