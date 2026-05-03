# Redis Connection Pool Tuning

Configure Redis client connection limits via `REDIS_POOL_SIZE` to match your workload.

## What REDIS_POOL_SIZE does

`REDIS_POOL_SIZE` sets the `--maxclients` argument passed to `redis-server` at container
start. Redis rejects new connections once this limit is reached, so the value must be at
least as large as the sum of all client pool sizes connecting to it.

| Environment | Default |
|---|---|
| `dev` | 20 |
| `staging` / `prod` | 50 |

The default is applied by `nself build` when the var is unset.

## When to tune

Increase `REDIS_POOL_SIZE` when:

- You see `ERR max number of clients reached` in Redis logs (`nself logs redis`).
- You have multiple plugins that use Redis (claw, queue, realtime, notify) and each has its
  own connection pool.
- Worker processes are horizontally scaled (more replicas = more connections).

Decrease it when running on a memory-constrained server — each idle client slot uses a small
amount of RAM (~20 KB per connection in Redis 7).

## Setting the value

Add to your `.env.local` or `.env.prod`:

```bash
# Total concurrent connections across all clients.
# Rule of thumb: sum of all plugin pool sizes + 10% headroom.
REDIS_POOL_SIZE=100
```

Then rebuild:

```bash
nself build
nself restart redis
```

## Sizing guide

| Workload | Suggested REDIS_POOL_SIZE |
|---|---|
| Single app, no plugins | 20 |
| claw + queue plugins | 50 (default prod) |
| claw + queue + realtime + notify | 100 |
| High-traffic multi-tenant | 200+ |

## Checking current limit

```bash
nself exec redis redis-cli CONFIG GET maxclients
```

## Related

- `REDIS_MEMORY` — container memory cap
- `REDIS_CPU` — container CPU cap
- [[cmd-build]] — regenerates `docker-compose.yml` after config changes
- [[cmd-logs]] — `nself logs redis` to inspect connection errors

[[Home]]
