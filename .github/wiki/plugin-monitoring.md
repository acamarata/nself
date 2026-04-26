# Monitoring Plugin

> Full observability stack with Prometheus, Grafana, and Loki. **Free, MIT licensed.**

## Install

```bash
nself plugin install monitoring
```

## What It Does

Installs 10 monitoring sub-services that provide metrics collection, log aggregation, distributed tracing, and alerting. Grafana dashboards are pre-configured for Postgres, Redis, and all ɳSelf core services. Access dashboards via subdomains on your base domain.

## Sub-Services

| Service | Image | Port | Purpose |
|---------|-------|------|---------|
| Prometheus | prom/prometheus | 9090 | Metrics collection and storage |
| Grafana | grafana/grafana | 3001 | Dashboard UI |
| Loki | grafana/loki:2.9.0 | 3100 | Log aggregation |
| Promtail | grafana/promtail:2.9.0 | — | Log shipping to Loki |
| Tempo | grafana/tempo | 3200 | Distributed tracing |
| Alertmanager | prom/alertmanager | 9093 | Alert routing and notifications |
| cAdvisor | gcr.io/cadvisor/cadvisor | 8082 | Container metrics |
| Node Exporter | prom/node-exporter | 9100 | Host metrics |
| Postgres Exporter | prometheuscommunity/postgres-exporter | 9187 | Postgres metrics |
| Redis Exporter | oliver006/redis_exporter | 9121 | Redis metrics (requires Redis) |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `GRAFANA_PORT` | `3001` | Grafana UI port |
| `PROMETHEUS_PORT` | `9090` | Prometheus port |
| `ALERTMANAGER_PORT` | `9093` | Alertmanager port |

## Ports

| Port | Purpose |
|------|---------|
| 9090 | Prometheus |
| 3001 | Grafana |
| 3100 | Loki |
| 9093 | Alertmanager |
| 8082 | cAdvisor |
| 9100 | Node Exporter |
| 9187 | Postgres Exporter |
| 9121 | Redis Exporter (profiles: [redis]) |

## Database Tables

0 tables, monitoring services store data in their own volumes.

## Nginx Routes

| Route | Target |
|-------|--------|
| `grafana.{base_domain}` | Grafana UI |
| `prometheus.{base_domain}` | Prometheus |
| `alertmanager.{base_domain}` | Alertmanager |
