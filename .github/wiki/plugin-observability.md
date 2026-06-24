# Observability Plugin

> Service health probes, watchdog timers, and service discovery with a unified health endpoint. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install observability
```

## What It Does

Runs configurable health probes against all registered services and exposes a unified `/health/` endpoint aggregating their status. A background watchdog monitors probe results and automatically restarts unhealthy services to reduce manual intervention. Service discovery registration allows dynamic addition of new services without restarting the plugin.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `OBSERVABILITY_PLUGIN_PORT` | `3215` | Port the Observability plugin service listens on |
| `OBSERVABILITY_CHECK_INTERVAL` | `30s` | Interval between health probe checks |
| `OBSERVABILITY_WATCHDOG_ENABLED` | `false` | Enable automatic restart of unhealthy services |

## Ports

| Port | Purpose |
|------|---------|
| `3215` | Observability plugin HTTP service |

## Database Tables

3 tables added to your Postgres database.

- `np_observability_services`, Registered services for discovery and probing
- `np_observability_health_history`, Health probe definitions and latest results
- `np_observability_watchdog_events`, Watchdog and state-change event log

## Nginx Routes

| Route | Description |
|-------|-------------|
| `/health/` | Aggregated health status endpoint, proxied to Observability plugin on port 3215 |

## See Also

- [[Home]] — nSelf CLI wiki homepage
- [[Plugin-Overview]] — All nSelf plugins
- [[Architecture]] — System architecture guide
