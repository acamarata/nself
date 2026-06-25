# Event Bus Plugin

> Internal event bus with pub/sub, fan-out delivery, dead-letter queue, and replay for inter-plugin messaging. Backed by NATS JetStream (embedded), Redpanda, or Kafka. **Pro plugin, requires license.**

> **Requires:** Pro license or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install event-bus
nself build
```

The license is validated against `ping.nself.org/license/validate`. An insufficient tier returns an error and the purchase URL.

## What It Does

The event-bus plugin provides a reliable internal messaging layer for other plugins and application services within a single nSelf deployment. It is **not** a customer-facing event system — it is infrastructure for inter-plugin communication.

Key capabilities:

- **Pub/sub** — publish a JSON message to a named subject; all subscribers receive it
- **Fan-out delivery** — multiple consumers per subject, each with their own delivery cursor
- **Dead-letter queue (DLQ)** — undeliverable messages are parked in a DLQ subject for inspection
- **Replay** — subjects retain messages up to a configurable retention window; new consumers can replay history
- **Broker flexibility** — defaults to an embedded NATS JetStream server (zero extra infra); swap to external NATS, Redpanda, or Kafka via env vars

## Subscribe (HTTP API)

Consumers subscribe using the NATS client library (embedded broker) or their broker's native client. For the embedded NATS broker:

```typescript
import { connect, StringCodec } from 'nats'

const nc = await connect({ servers: 'nats://your-nself-host:4222' })
const sc = StringCodec()
const js = nc.jetstream()

// Subscribe to a subject
const sub = await js.subscribe('events.users.created', {
  durable: 'my-consumer',
  ack_policy: 'explicit',
})

for await (const msg of sub) {
  const payload = sc.decode(msg.data)
  console.log('received:', payload)
  msg.ack()
}
```

## Publish (HTTP Control Plane)

The plugin exposes an HTTP control plane for publishing, status, and administration:

```bash
# Publish a message
curl -X POST http://your-nself-host:8212/event-bus/publish \
  -H 'Content-Type: application/json' \
  -d '{"subject":"events.users.created","payload":{"user_id":"abc123"}}'

# Check broker status
curl http://your-nself-host:8212/event-bus/status

# List active subjects
curl http://your-nself-host:8212/event-bus/subjects

# List consumers
curl http://your-nself-host:8212/event-bus/consumers

# Purge a subject's stats
curl -X POST http://your-nself-host:8212/event-bus/purge/events.users.created
```

## Dead-Letter Queue

Messages that fail delivery after all retry attempts are routed to `$JS.EVENT.DLQ.<subject>`. Inspect and replay:

```bash
# List DLQ messages via NATS CLI
nats stream get EVENTS --seq 42
```

Configure max delivery attempts with the `MAX_DELIVER` JetStream consumer setting.

## Broker Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | Postgres connection string (stats persistence) |
| `EVENT_BUS_PORT` | No | `8212` | HTTP control plane port |
| `EVENT_BUS` | No | `nats` | Broker type: `nats` \| `redpanda` \| `kafka` |
| `NATS_EMBEDDED` | No | `true` | Start embedded NATS server |
| `NATS_URL` | No | — | External NATS URL (when `NATS_EMBEDDED=false`) |
| `NATS_CLIENT_PORT` | No | `4222` | Embedded NATS client port |
| `NATS_STORE_DIR` | No | `/tmp/nats-store` | Embedded NATS JetStream storage |
| `RETENTION_MS` | No | `86400000` | Message retention window (ms) |
| `MAX_BYTES` | No | `1073741824` | Stream max size (bytes) |
| `REDPANDA_BROKERS` | No | — | Redpanda broker list (comma-separated) |
| `KAFKA_BROKERS` | No | — | Kafka broker list (comma-separated) |
| `KAFKA_SASL_USER` | No | — | Kafka SASL username |
| `KAFKA_SASL_PASS` | No | — | Kafka SASL password |

## Postgres Schema

| Table | Purpose |
|-------|---------|
| `np_event_bus_stats` | Per-subject message count and consumer count (internal metrics) |

The stats table includes `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation (Convention A). Stats are flushed from memory to Postgres periodically.

## HTTP Routes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/event-bus/status` | Broker status + subject count |
| `GET` | `/event-bus/subjects` | List tracked subjects with message counts |
| `POST` | `/event-bus/publish` | Publish a message to a subject |
| `GET` | `/event-bus/consumers` | List active consumers |
| `POST` | `/event-bus/purge/{subject}` | Reset stats for a subject |

## Port

This plugin runs on port `8212` (CS_9 slot per the nSelf port registry).

## Bundle

This plugin is not included in a named bundle. Access is granted with any Pro license tier or ɳSelf+.

## See Also

- [Architecture: Microkernel](Architecture-Microkernel.md)
- [Plugin Reference](Plugin-Reference.md)
