# nself project

Manage ɳSelf projects under the multi-tenant master controller.

Each project is fully isolated: dedicated Postgres schema, Hasura metadata source, Nginx virtual host, JWT secret, Redis key prefix, and MinIO bucket.

**Requires:** `NSELF_FLAG_MULTI_TENANT_CONTROLLER=true`

## Synopsis

```
nself project <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
|---|---|
| `create` | Provision a new isolated project |
| `delete` | Delete a project (irreversible) |
| `list` | List all projects |
| `status` | Show health and resource stats |
| `migrate` | Run pending migrations |
| `shell` | Open psql scoped to the project schema |
| `rotate-credentials` | Rotate role password and JWT secret |

---

## nself project create

Provisions a new project in under 10 seconds (p50 target: 4s).

```
nself project create --slug <slug> --domain <domain>
```

**Steps:** schema + role provisioning, Hasura source registration, Nginx vhost generation, JWT secret generation.

**Example:**

```bash
nself project create --slug myapp --domain myapp.myserver.com
# Project ready: https://myapp.myserver.com (id: abc123...)
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--slug` | Yes | Project identifier (lowercase, alphanumeric, hyphens, 3-40 chars) |
| `--domain` | Yes | Full domain for the project (must be DNS-resolvable) |

---

## nself project delete

Deletes a project and removes all associated resources. **Irreversible.**

```
nself project delete --slug <slug> --force
```

**Teardown sequence:** Hasura source deregistered, Nginx vhost removed, Redis keys flushed, MinIO bucket dropped, Postgres schema + role dropped, audit log written.

### Flags

| Flag | Required | Description |
|---|---|---|
| `--slug` | Yes | Project slug to delete |
| `--force` | Yes | Confirm irreversible deletion |

---

## nself project list

Lists all projects managed by the controller.

```
nself project list
```

**Output:**

```
SLUG    DOMAIN                  STATUS  CREATED
myapp   myapp.myserver.com      active  2026-04-23T10:00:00Z
```

---

## nself project status

Shows health and resource stats for a single project: connection count, schema size, last active.

```
nself project status --slug <slug>
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--slug` | Yes | Project slug |

---

## nself project migrate

Runs pending database migrations for a project schema. Delegates to `nself deploy` with the project schema targeted.

```
nself project migrate --slug <slug>
```

---

## nself project shell

Opens a psql shell authenticated as the per-project role, with `search_path` set to the project schema.

```
nself project shell --slug <slug>
```

---

## nself project rotate-credentials

Rotates the project role password and JWT secret. Updates the project `.env` file and Hasura source connection string.

```
nself project rotate-credentials --slug <slug>
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `NSELF_FLAG_MULTI_TENANT_CONTROLLER` | `false` | Must be `true` |
| `NSELF_CONTROLLER_ADDR` | `http://127.0.0.1:3750` | Controller daemon address |
| `NSELF_CONTROLLER_ADMIN_TOKEN` | — | Admin token (required) |
| `NSELF_CONTROLLER_MAX_PROJECTS` | `50` | Hard cap on projects |
| `NSELF_PROJECT_SCHEMA_PREFIX` | `project_` | Postgres schema prefix |
| `NSELF_PROJECT_REDIS_PREFIX` | `proj:` | Redis key prefix |
| `NSELF_PROJECT_HASURA_PORT_START` | `8080` | First port for Hasura source routing |

## Multi-Tenant Convention Wall

B46 uses `tenant_id UUID` for Cloud SaaS isolation, NOT `source_account_id`. These are distinct mechanisms. See the [Multi-Tenant Convention Wall](../../docs/architecture/multi-tenant-conventions.md).

## See Also

- [`nself controller`](controller.md), Daemon management
- [Multi-Tenant Hosting Guide](../../docs/guides/multi-tenant-hosting.md)
- [Agency Setup Guide](../../docs/guides/agency-setup.md)
