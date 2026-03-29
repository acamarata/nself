# Feature: Monitoring

The nSelf monitoring stack provides metrics, logs, traces, and alerting for your entire backend infrastructure.

## What's Included

Install with `nself plugin install monitoring`. Adds 10 services:

| Service | Port | Purpose |
|---------|------|---------|
| Prometheus | 9090 | Metrics collection and storage |
| Grafana | (see `nself urls`) | Dashboards and visualisation |
| Loki | 3100 | Log aggregation |
| Promtail | — | Log shipping agent |
| Tempo | 3200 | Distributed tracing |
| Alertmanager | 9093 | Alert routing |
| cAdvisor | 8081 | Container resource metrics |
| Node Exporter | 9100 | Host OS metrics |
| Postgres Exporter | 9187 | Database metrics |
| Redis Exporter | 9121 | Redis metrics |

## Quick Start

```bash
nself plugin install monitoring
nself build && nself restart
nself urls    # get Grafana URL
```

## Pre-Built Dashboards

| Dashboard | Data Source |
|-----------|------------|
| Container Resources | cAdvisor |
| Host Metrics | Node Exporter |
| PostgreSQL | Postgres Exporter |
| Nginx | Nginx logs via Promtail |
| nSelf Overview | Combined |

## Log Aggregation

All container stdout/stderr is shipped to Loki via Promtail. Query in Grafana:

```logql
{container=~"myproject_.*"} |= "error" | json
```

## Alerts

Pre-built alert rules in Alertmanager cover:
- Container down
- High CPU / memory
- Disk space low
- PostgreSQL connection pool exhausted

Configure notification channels (Slack, PagerDuty, email) in `monitoring/alertmanager/config.yml`.

## See Also

- [[Guide-Monitoring-Setup]] — step-by-step setup guide
- [[plugin-monitoring]] — monitoring plugin reference

---
← [[Home]] | [[_Sidebar]]
