# Guide: Monitoring Setup

nSelf's monitoring stack provides metrics, logs, and alerting for your entire backend.

## Install the Monitoring Plugin

```bash
nself plugin install monitoring
nself build && nself restart
```

This adds 10 services to your stack:

| Service | Purpose |
|---------|---------|
| Prometheus | Metrics collection and storage |
| Grafana | Metrics dashboards |
| Loki | Log aggregation |
| Promtail | Log shipping agent |
| Tempo | Distributed tracing |
| Alertmanager | Alert routing and notifications |
| cAdvisor | Container resource metrics |
| Node Exporter | Host OS metrics |
| Postgres Exporter | Database metrics |
| Redis Exporter | Redis metrics (if Redis enabled) |

## Access Grafana

```bash
nself urls    # shows Grafana URL and default credentials
```

Default credentials are in `.env` as `GRAFANA_USER` / `GRAFANA_PASSWORD`. Change them immediately.

## Pre-Built Dashboards

Grafana comes pre-loaded with dashboards for:

- **Container metrics** — CPU, memory, network per container (cAdvisor)
- **Host metrics** — disk, CPU, memory, load (Node Exporter)
- **PostgreSQL** — connections, query rates, cache hit ratio (Postgres Exporter)
- **Nginx** — request rates, error rates, response times

## Log Aggregation with Loki

All container logs are shipped to Loki via Promtail. Query logs in Grafana using LogQL:

```logql
{container="myproject_hasura_1"} |= "error"
```

## Setting Up Alerts

Edit `monitoring/alertmanager/config.yml` to configure notification channels:

```yaml
receivers:
  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/...'
        channel: '#alerts'
```

Pre-built alert rules cover:
- Container down
- High CPU / memory usage
- Postgres connection exhaustion
- Disk space low

## See Also

- [[Guide-Production-Deployment]] — initial server setup
- [[plugin-monitoring]] — monitoring plugin reference
- [[Feature-Monitoring]] — monitoring feature overview

---
← [[Home]] | [[_Sidebar]]
