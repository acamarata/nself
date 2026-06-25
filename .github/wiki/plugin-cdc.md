# CDC Plugin (Pro)

> Change Data Capture — streams Postgres WAL events to Kafka, Redpanda, or RabbitMQ using Debezium envelope format. **Pro plugin.**

> **Requires:** Pro license or higher. `nself license set nself_pro_...`
>
> **Custom Service Slot:** CS_5 — CDC runs as a custom service slot alongside the standard nSelf plugin set.

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install cdc
```

## What It Does

Tails the Postgres WAL (Write-Ahead Log) using logical replication and streams INSERT/UPDATE/DELETE events to downstream message brokers. Events are emitted in a Debezium-compatible envelope so existing Debezium consumers work without changes.

Use this plugin to:
- Replicate database changes to a data lake or analytics platform.
- Build event-driven microservices that react to Postgres changes.
- Maintain a full audit trail of all database mutations in an external system.
- Sync Postgres state to Elasticsearch, Redis, or other secondary stores.

## Prerequisites

Your Postgres instance must have logical replication enabled:
```sql
-- postgresql.conf
wal_level = logical
max_replication_slots = 5
max_wal_senders = 5
```

Create the replication publication (nSelf does this automatically on install):
```sql
CREATE PUBLICATION nself_pub FOR ALL TABLES;
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | — | Postgres connection string with replication privilege (required) |
| `CDC_PORT` | `8209` | CDC HTTP health/management port |
| `CDC_BROKER` | `kafka` | Broker type: `kafka`, `redpanda`, or `rabbitmq` |
| `CDC_BROKER_URLS` | — | Comma-separated broker addresses (e.g. `kafka:9092`) |
| `CDC_TOPIC_PREFIX` | `nself` | Topic/exchange name prefix |
| `CDC_SLOT_NAME` | `nself_cdc` | Postgres replication slot name |
| `CDC_PUBLICATION_NAME` | `nself_pub` | Postgres publication name |
| `CDC_BATCH_SIZE` | `100` | Events per batch before flush |
| `CDC_FLUSH_MS` | `50` | Max milliseconds before forced flush |

## Ports

| Port | Purpose |
|------|---------|
| 8209 | CDC health check and management API |

## Custom Service Slot (CS_5)

CDC uses the `CS_5` custom service slot in the nSelf service inventory. This means it runs as a long-lived background process outside the standard plugin HTTP server model. The slot is reserved for CDC exclusively — do not assign another plugin to CS_5.

## Broker Support

| Broker | `CDC_BROKER` value | Protocol |
|--------|-------------------|---------|
| Apache Kafka | `kafka` | Kafka protocol |
| Redpanda | `redpanda` | Kafka protocol |
| RabbitMQ | `rabbitmq` | AMQP 0-9-1 |

## Event Envelope

Events follow the Debezium envelope format:

```json
{
  "schema": { "type": "struct", "name": "np_users.Envelope" },
  "payload": {
    "op": "c",
    "before": null,
    "after": { "id": "...", "email": "..." },
    "source": {
      "connector": "nself-cdc",
      "table": "np_users",
      "ts_ms": 1719340000000
    }
  }
}
```

Operations: `c` = create (INSERT), `u` = update (UPDATE), `d` = delete (DELETE), `r` = read (snapshot).

## API

```
GET /health     — Liveness check
GET /status     — Replication slot lag and broker connection status
POST /pause     — Pause WAL consumption
POST /resume    — Resume WAL consumption
```
