# Access Controls Plugin

> RBAC + ABAC with policy engine and permission cache. **Pro plugin** — requires a license.

> **Requires:** Any paid bundle or ɳSelf+ license. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install access-controls
```

## What It Does

Extends ɳSelf with fine-grained authorization. Implements both Role-Based Access Control (RBAC) and Attribute-Based Access Control (ABAC). Define roles, assign permissions to those roles, and write ABAC policy rules that evaluate user attributes, resource properties, and request context. Authorization results are cached in-process with a configurable TTL for fast permission checks on every request.

### RBAC vs ABAC

**RBAC** is best for straightforward, role-centric permission models.

- Create roles (e.g., `admin`, `editor`, `viewer`) with parent-child hierarchy.
- Assign permissions (resource + action pairs) to roles.
- Assign roles to users, optionally scoped to a resource or channel.

**ABAC** handles conditional access that cannot be expressed in role terms alone.

- Write policies that fire when user attributes, resource properties, and context match a condition.
- Example: allow a user to delete a post only when `user_id == post.author_id`.
- Policies use `$eq`, `$ne`, `$in`, `$gt`, `$lt`, and `$lte` condition operators.
- Both RBAC and ABAC can be combined: RBAC grants baseline access; ABAC refines edge cases.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ACL_PLUGIN_PORT` | `3027` | Service port |
| `ACL_CACHE_TTL_SECONDS` | `300` | In-process permission cache TTL in seconds |
| `ACL_DEFAULT_DENY` | `true` | Deny access when no policy matches (recommended) |
| `ACL_MAX_ROLE_DEPTH` | `10` | Maximum role inheritance depth |
| `ACL_API_KEY` | — | Inter-plugin auth key (auto-provided by nself) |
| `ACL_RATE_LIMIT_MAX` | `100` | Max requests per rate-limit window |
| `ACL_RATE_LIMIT_WINDOW_MS` | `60000` | Rate-limit window in milliseconds |

All `DATABASE_URL` and API key vars are injected automatically by the nself CLI.

## Ports

| Port | Purpose |
|------|---------|
| 3027 | Access controls REST API |

## Database Tables

Six tables are added to your Postgres database using the `np_acl_` prefix:

| Table | Purpose |
|-------|---------|
| `np_acl_roles` | Role definitions with hierarchy |
| `np_acl_permissions` | Permission definitions (resource + action) |
| `np_acl_role_permissions` | Role to permission mappings |
| `np_acl_user_roles` | User to role assignments |
| `np_acl_policies` | ABAC policy rules |
| `np_acl_webhook_events` | Event and audit log |

All tables include `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation.

## Hasura Integration

The access-controls service exposes a REST API at port 3027. Integrate with Hasura in two ways:

**Row-level permission checks via Remote Schema action:** Call `POST /v1/authorize` from a Hasura action to gate mutations. Pass `user_id`, `resource`, `action`, and optional `context` attributes.

**Hasura row filter pattern:** For data tables that rely on role-based access, configure Hasura row filters to pass `X-Hasura-User-Id` in request context, then call the `/v1/authorize` endpoint in a webhook or event trigger to enforce ABAC conditions before returning data.

All `np_acl_*` tables are accessible via Hasura GraphQL with `source_account_id` row-level filtering enforced for multi-app deployments.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/access-controls/` | Access controls management API |
| `/access-controls/check` | Permission check endpoint |

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/authorize` | Check if user has permission |
| `POST` | `/v1/authorize/batch` | Batch authorization check |
| `POST` | `/v1/roles` | Create a role |
| `GET` | `/v1/roles` | List roles |
| `GET` | `/v1/roles/hierarchy` | Get role hierarchy tree |
| `POST` | `/v1/users/:userId/roles` | Assign role to user |
| `GET` | `/v1/users/:userId/permissions` | List effective permissions |
| `POST` | `/v1/policies` | Create ABAC policy |
| `GET` | `/health` | Health check |

## Related

- [[Plugin-Overview]] — all plugins at a glance
- [[cmd-plugin]] — `nself plugin` command reference
- [[Home]]
