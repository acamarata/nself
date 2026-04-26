# Access Controls Plugin

> RBAC + ABAC with policy engine and permission cache. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install access-controls
```

## What It Does

Extends ɳSelf Auth with fine-grained access control. Implements both Role-Based Access Control (RBAC) and Attribute-Based Access Control (ABAC). Define roles, permissions, and policy rules that evaluate user attributes, resource properties, and context. Results are cached in Redis for fast permission checks on every request. Integrates with Hasura's permission system.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ACCESS_CONTROLS_PORT` | `3027` | Access controls service port |
| `ACCESS_CONTROLS_CACHE_TTL` | `300` | Permission cache TTL in seconds |
| `ACCESS_CONTROLS_DEFAULT_DENY` | `true` | Deny access if no policy matches |
| `ACCESS_CONTROLS_AUDIT_ENABLED` | `true` | Log all permission decisions |

## Ports

| Port | Purpose |
|------|---------|
| 3027 | Access controls REST API |

## Database Tables

6 tables added to your Postgres database:
- `np_access_controls_roles`, role definitions
- `np_access_controls_permissions`, permission definitions
- `np_access_controls_role_permissions`, role-permission mappings
- `np_access_controls_policies`, ABAC policy rules
- `np_access_controls_assignments`, user-role assignments
- `np_access_controls_audit`, permission decision log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/access-controls/` | Access controls management API |
| `/access-controls/check` | Permission check endpoint |
