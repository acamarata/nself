# Feature: Monitoring

The ɳSelf monitoring stack provides metrics, logs, traces, and alerting for your entire backend infrastructure. It is a **free, first-class bundle** included in every nSelf install. No license key is required.

See [[bundle-monitoring]] for the full bundle reference including the difference from the paid ɳSentry bundle.

## What's Included

The Monitoring bundle ships with 10 services, configured automatically by `nself build`:

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
nself monitoring start     # start all 10 monitoring services
nself monitoring status    # verify running state + URLs
nself urls                 # get Grafana URL
```

## Pre-Built Dashboards

| Dashboard | Data Source |
|-----------|------------|
| Container Resources | cAdvisor |
| Host Metrics | Node Exporter |
| PostgreSQL | Postgres Exporter |
| Nginx | Nginx logs via Promtail |
| ɳSelf Overview | Combined |

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

- [[bundle-monitoring]], Monitoring bundle reference (free, always included, no license required)
- [[Guide-Monitoring-Setup]], step-by-step setup guide

---
← [[Home]] | [[_Sidebar]]
