# Monitoring

nSelf includes a full monitoring stack that runs alongside your backend. It is free, included in every install, and requires no configuration.

## What's included

| Service | Port | Purpose |
|---------|------|---------|
| Prometheus | 9090 | Metrics collection |
| Grafana | 3000 (internal) | Dashboards |
| Loki | 3100 | Log aggregation |
| Promtail | — | Log shipper (runs as sidecar) |
| Tempo | — | Distributed tracing |
| Alertmanager | 9093 | Alert routing |
| cAdvisor | 8080 | Container metrics |
| Node Exporter | 9100 | Host metrics |
| Postgres Exporter | 9187 | Database metrics |
| Redis Exporter | 9121 | Cache metrics |

All monitoring services bind to 127.0.0.1. Access via Nginx proxy at `monitor.<your-domain>` or `localhost:3000` (Grafana) during local dev.

## Enable monitoring

Monitoring is enabled by default. To disable:

```bash
nself config set monitoring.enabled false
nself build  # regenerates docker-compose without monitoring stack
```

## Accessing Grafana

```bash
nself admin open monitoring
# Opens http://localhost:3000 in your browser
# Default credentials: admin / (generated at install, see .env.secrets GRAFANA_PASSWORD)
```

## Pre-built dashboards

Grafana includes dashboards for:
- **System Overview** — CPU, memory, disk, network
- **Postgres** — query performance, connections, cache hit ratio
- **Hasura** — request rate, error rate, latency
- **Container resources** — per-service CPU and memory
- **Application logs** — via Loki data source

## Custom alerts

Alerts are configured in `nself/alertmanager/config.yml`. After editing:

```bash
nself build
nself start
```

## Monitoring in production

The monitoring stack runs on the same VPS as your nSelf instance. For the nSelf Cloud tier, monitoring data is stored in isolated per-tenant volumes.

## Troubleshooting

**Prometheus not scraping a service:**
```bash
nself logs prometheus --tail 50
```

**Grafana can't connect to Loki:**
```bash
nself doctor --section monitoring
```
