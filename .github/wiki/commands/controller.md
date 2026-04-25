# nself controller

Manage the nCloud multi-tenant master controller.

The controller manages N isolated nSelf project instances on a single server. Each project gets its own Postgres schema, Hasura metadata source, Nginx virtual host, JWT secret, Redis key prefix, and MinIO bucket.

**Requires:** `NSELF_FLAG_MULTI_TENANT_CONTROLLER=true`

## Synopsis

```
nself controller <subcommand>
```

## Subcommands

| Subcommand | Description |
|---|---|
| `start` | Start the controller daemon |
| `stop` | Stop the controller daemon |
| `status` | Show all projects and resource usage |

## nself controller start

Starts the controller daemon as a systemd service (`nself-controller`). On non-systemd systems, prints the command to run the binary directly.

```
nself controller start
```

## nself controller stop

Stops the controller daemon.

```
nself controller stop
```

## nself controller status

Displays all projects with their resource usage: connection count, schema size, and health.

```
nself controller status
```

**Output:**

```
SLUG         DOMAIN                    STATUS  CONNS  SIZE    HEALTHY
myapp        myapp.myserver.com        active  2      1.2 MB  true
clientb      clientb.myserver.com      active  0      512 KB  true

2 / 50 projects active (4% utilized)
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `NSELF_FLAG_MULTI_TENANT_CONTROLLER` | `false` | Enable multi-tenant controller |
| `NSELF_CONTROLLER_ADDR` | `http://127.0.0.1:3750` | Controller daemon address |
| `NSELF_CONTROLLER_ADMIN_TOKEN` | — | Admin authentication token (required) |
| `NSELF_CONTROLLER_HOST` | `127.0.0.1` | Daemon bind host |
| `NSELF_CONTROLLER_PORT` | `3750` | Daemon bind port |

## See Also

- [`nself project`](project.md) — Manage individual projects
- [Multi-Tenant Hosting Guide](../../docs/guides/multi-tenant-hosting.md)
