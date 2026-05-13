# Monitoring Bundle

## Contents

- [What Is the Monitoring Bundle](#what-is-the-monitoring-bundle)
- [Services Included](#services-included)
- [Always Free](#always-free)
- [Commands](#commands)
- [Access URLs](#access-urls)
- [Difference from ɳSentry](#difference-from-nsentry)
- [Related Pages](#related-pages)

## What Is the Monitoring Bundle

The Monitoring bundle ships with every nSelf install at no cost. It provides infrastructure observability: metrics, logs, distributed traces, and alerts for your entire backend stack — Postgres, Redis, containers, and the host OS.

No license key is required. No bundle purchase is needed. The Monitoring bundle is part of the nSelf free core.

## Services Included

10 services, all free, all automatically configured by `nself build`.

| Service | Port | Purpose |
|---------|------|---------|
| Prometheus | 9090 | Metrics collection and storage |
| Grafana | (see `nself urls`) | Dashboards and visualisation |
| Loki | 3100 | Log aggregation |
| Promtail | — | Log shipping agent (runs as sidecar) |
| Tempo | 3200 | Distributed tracing |
| Alertmanager | 9093 | Alert routing (Slack, PagerDuty, email) |
| cAdvisor | 8081 | Container resource metrics |
| Node Exporter | 9100 | Host OS metrics (CPU, memory, disk, network) |
| Postgres Exporter | 9187 | Database metrics (connections, queries, replication) |
| Redis Exporter | 9121 | Redis metrics (hit rate, memory, commands) |

## Always Free

The Monitoring bundle is a first-class free bundle. It differs from paid plugin bundles in three ways:

- No `nself license set` step required.
- Included in every `nself init` + `nself build` run.
- Cannot be separated from the core stack — it is always present.

To verify monitoring is running:

```bash
nself monitoring status
```

## Commands

```bash
nself monitoring start     # start all 10 monitoring services
nself monitoring stop      # stop all 10 monitoring services
nself monitoring status    # show running state + URLs
nself urls                 # print all service URLs, including Grafana
```

Pre-built dashboards cover containers (cAdvisor), host OS (Node Exporter), Postgres, Nginx, and a combined ɳSelf overview. Alertmanager rules for container-down, high CPU/memory, low disk, and connection pool exhaustion are included by default.

## Access URLs

| Environment | Grafana URL |
|-------------|-------------|
| Local dev | `http://localhost:3000` |
| Deployed | `https://grafana.<your-domain>` |

Run `nself urls` after `nself start` to get the exact Grafana URL for your current environment.

All monitoring services bind to `127.0.0.1`. External access goes through Nginx, which is configured automatically by `nself build`. Never expose Prometheus or Grafana ports directly to the public internet.

## Difference from ɳSentry

nSelf ships two distinct observability layers. They do not overlap.

| | Monitoring bundle | ɳSentry bundle |
|---|---|---|
| **Scope** | Infrastructure: containers, host OS, Postgres, Redis | Product/app: uptime, incidents, error tracking, SLOs, RUM |
| **Cost** | Free, always included | $0.99/mo (or ɳSelf+ $3.99/mo) |
| **License required** | No | Yes |
| **Example questions answered** | "Is my Postgres running? CPU usage? Disk free?" | "Is my app up for users? Error rate? Alert me when it goes down." |
| **Services** | Prometheus, Grafana, Loki, Promtail, Tempo, Alertmanager, cAdvisor, Node Exporter, Postgres Exporter, Redis Exporter | nself-uptime-monitor, nself-status-page, nself-incident-mgmt, nself-alert-router, nself-slo-tracker, nself-synthetic-monitor, nself-rum, and more |

Use both layers together for complete observability. The Monitoring bundle is always running. ɳSentry is optional and adds product-level visibility on top.

## Related Pages

- [[Feature-Monitoring]] — step-by-step monitoring setup guide
- [[bundle-nclaw]] — ɳClaw bundle (paid, $0.99/mo)
- [[bundle-nchat]] — ɳChat bundle (paid, $0.99/mo)
- [[Config-Optional-Services]] — optional services configuration

---
← [[Home]] | [[_Sidebar]]
