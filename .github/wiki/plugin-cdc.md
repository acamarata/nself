# Plugin: cdc — Change Data Capture

Streams Postgres WAL change events (INSERT / UPDATE / DELETE) to Kafka, Redpanda, or RabbitMQ using a Debezium-compatible envelope format. Designed for event-driven architectures that need a live feed of database mutations without polling.

**Port:** 8209 · **Tier:** Pro · **Custom service slot:** CS_5 · **Status:** ✅ GA

---

## Requirements

- nSelf v1.1.0+
- A valid Pro or ɳSelf+ license
- Postgres with `wal_level = logical` (see WAL Configuration below)
- Docker-accessible Kafka, Redpanda, or RabbitMQ endpoint

---

## Install

```bash
nself license set nself_pro_<your-key>
nself plugin install cdc
```

The CLI pulls `nself-org/plugin-cdc:latest` from Docker Hub, wires it into the CS_5 slot, and runs the database migration automatically.

---

## WAL Configuration

Logical replication must be enabled in Postgres before the plugin starts. Verify:

```sql
SHOW wal_level;  -- must return 'logical'
```

If not already set, add to `postgresql.conf` (or pass as a startup flag) and restart Postgres:

```
wal_level = logical
max_replication_slots = 5
max_wal_senders = 5
```

On managed providers (RDS, Supabase, Neon):

- **RDS:** set `rds.logical_replication = 1` in the parameter group and reboot.
- **Supabase:** logical replication is on by default for Pro/Team plans.
- **Neon:** enable via the console → Project → Settings → Logical Replication.

The migration creates a replication slot (`nself_cdc`) and a publication (`nself_pub FOR ALL TABLES`) automatically on first start.

---

## Environment Variables

### Required

| Variable | Description |
|---|---|
| `DATABASE_URL` | Postgres connection string with a role that has `REPLICATION` privilege |
| `CDC_BROKER` | Broker type: `kafka`, `redpanda`, or `rabbitmq` |
| `CDC_BROKER_URLS` | Comma-separated broker addresses, e.g. `kafka:9092,kafka2:9092` |

### Optional

| Variable | Default | Description |
|---|---|---|
| `CDC_TOPIC_PREFIX` | `nself` | Prefix for all Kafka topics / RabbitMQ routing keys |
| `CDC_SLOT_NAME` | `nself_cdc` | Postgres logical replication slot name |
| `CDC_PUBLICATION_NAME` | `nself_pub` | Postgres publication name |
| `CDC_TABLES` | *(all np_* tables)* | Comma-separated table allowlist |
| `CDC_MODE` | `embedded` | `embedded` (built-in WAL reader) or `debezium-connect` |
| `CDC_DEBEZIUM_URL` | *(unset)* | Debezium Connect REST URL when mode = `debezium-connect` |
| `CDC_BATCH_SIZE` | `100` | Events per broker batch |
| `CDC_FLUSH_MS` | `50` | Broker flush interval in milliseconds |
| `CDC_PORT` | `8209` | HTTP control-plane port (CS_5 slot) |

### Minimal configuration example

```env
DATABASE_URL=postgres://nself:password@db:5432/nself?sslmode=require
CDC_BROKER=kafka
CDC_BROKER_URLS=kafka:9092
```

---

## Topic / Routing Key Format

Events are published to topics named:

```
<CDC_TOPIC_PREFIX>.<table_name>.<operation>
```

Operation values: `insert`, `update`, `delete`

Examples:

```
nself.np_users.insert
nself.np_orders.update
nself.np_orders.delete
```

---

## Event Envelope (Debezium Format)

Every event is published as JSON conforming to the Debezium source envelope:

```json
{
  "op": "c",
  "before": null,
  "after": {
    "id": "42",
    "email": "alice@example.com"
  },
  "ts_ms": 1714000000000,
  "source": {
    "table": "np_users",
    "lsn": "0/1A2B3C"
  }
}
```

| Field | Value |
|---|---|
| `op` | `c` = INSERT, `u` = UPDATE, `d` = DELETE |
| `before` | Row state before the change (null for inserts; populated when replica identity is FULL) |
| `after` | Row state after the change (null for deletes) |
| `ts_ms` | Event timestamp in milliseconds UTC |
| `source.lsn` | Postgres WAL log sequence number |

---

## Consumer Group Setup (Kafka)

Consumer groups receive events in-order per table. Set `auto.offset.reset = earliest` to replay from the beginning of the slot.

```go
r := kafka.NewReader(kafka.ReaderConfig{
    Brokers:     []string{"kafka:9092"},
    Topic:       "nself.np_orders.insert",
    GroupID:     "my-service-consumer",
    StartOffset: kafka.FirstOffset,
})
```

---

## CS_5 Custom Service Slot

CDC runs as a long-lived service in the **CS_5** slot — one of five custom service slots reserved for infrastructure plugins. This means:

- `nself start` and `nself stop` include the CDC service automatically.
- The service appears in `nself status` as `cdc (CS_5) ✅` when running.
- Health checks are polled at `/cdc/health` on port 8209.
- Slot CS_5 is dedicated to this plugin; do not assign another custom service to CS_5.

---

## HTTP Control Plane (port 8209)

| Method | Path | Description |
|---|---|---|
| `GET` | `/cdc/status` | Replication slot lag, events/s, broker connection state |
| `GET` | `/cdc/topics` | List of active topics with message counts |
| `POST` | `/cdc/snapshot?table=np_users` | Trigger full-table initial snapshot to broker |
| `POST` | `/cdc/pause` | Pause WAL streaming (slot remains open) |
| `POST` | `/cdc/resume` | Resume streaming and drain the back-pressure buffer |
| `DELETE` | `/cdc/slot?confirm=true` | Drop the replication slot (irreversible — use with care) |

---

## Back-Pressure and Buffering

When the broker is unavailable, events are written to `np_cdc_events` with `brokered = false`. On reconnect, the buffer is drained in order before live CDC resumes.

**Hard stop:** the WAL reader pauses automatically when the unbrokered buffer exceeds 100 000 rows. Resume the broker connection; `POST /cdc/resume` to restart.

---

## Uninstall

```bash
nself plugin uninstall cdc
```

This drops the replication slot, removes the publication, and stops the CS_5 service.

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `replication connect: no pg_hba.conf entry for replication` | Database role missing REPLICATION privilege | `ALTER ROLE nself REPLICATION;` |
| `slot reader: ensure slot: maximum number of replication slots reached` | Existing slots not cleaned up | `SELECT pg_drop_replication_slot(slot_name) FROM pg_replication_slots WHERE slot_name LIKE 'nself_%';` |
| `kafka: write N messages: dial tcp: connection refused` | Kafka unreachable at startup | Verify `CDC_BROKER_URLS` and network reachability; events buffer in `np_cdc_events` |
| Topics never appear in Kafka | `wal_level` is not `logical` | Set `wal_level = logical` and restart Postgres |
| Plugin exits immediately | Missing required env vars | Check `DATABASE_URL`, `CDC_BROKER`, `CDC_BROKER_URLS` |

---

## Further Reading

- [Postgres Logical Replication Docs](https://www.postgresql.org/docs/current/logical-replication.html)
- [Debezium Change Event Format](https://debezium.io/documentation/reference/stable/connectors/postgresql.html#postgresql-change-events-value)
- [nSelf Plugin Docs](https://docs.nself.org/plugins/cdc)
