# Event Bus Plugin

> Pro internal event bus — NATS JetStream-backed pub/sub, fan-out delivery, dead-letter queue, and message replay for inter-plugin messaging. Requires a pro license.

## Install

```bash
nself plugin install event-bus
nself build && nself start
```

The embedded NATS server starts automatically. No external broker is required by default.

## What It Does

Provides a shared messaging backbone across all installed plugins and custom services. Plugins publish events to named subjects and subscribe to receive them. Messages are durably stored in JetStream streams so consumers survive restarts. Fan-out means one published message reaches every active subscriber concurrently. Failed deliveries move to a dead-letter queue (DLQ) for inspection and replay.

The event bus is internal infrastructure — it is not exposed via Nginx and is not customer-facing. Other plugins (CDC, warehouse, webhooks) use it as a reliable event backbone.

## Broker Options

Set `EVENT_BUS` to choose your message broker:

| Value | Broker | Notes |
|-------|--------|-------|
| `nats` (default) | NATS JetStream | Embedded (≤50 MB RAM) or external |
| `redpanda` | Redpanda | Kafka-protocol compatible |
| `kafka` | Apache Kafka | SASL/PLAIN supported |

## Subject Naming

All subjects must start with `nself.`:

```
nself.<plugin>.<entity>.<operation>

# Examples
nself.cdc.np_users.insert
nself.claw.memory.update
nself.webhook.delivery.failed
nself.custom.<user-defined>
```

Wildcard subscriptions: `nself.cdc.*` receives every CDC event from all tables.

## Subscribe API

```go
import sdk "github.com/nself-org/nself-sdk"

bus, err := sdk.NewEventBus(sdk.EventBusConfig{
    Endpoint: "http://127.0.0.1:8212",
})

// Durable subscription — survives consumer restart
sub, err := bus.Subscribe("nself.cdc.*", func(msg sdk.EventBusMessage) error {
    // msg.Subject, msg.Payload available
    return msg.Ack()
})
```

## Publish API

```go
// Publish a JSON payload to a subject
err = bus.Publish(ctx, sdk.SubjectName("cdc", "np_users", "insert"), payload)
```

## Unsubscribe

```go
sub.Unsubscribe()
```

Durable consumers retain their position; calling `Unsubscribe` stops delivery but does not delete the consumer from JetStream. Delete the consumer explicitly if you want to reset the replay position.

## DLQ — Dead-Letter Queue

When a subscriber returns an error after all retry attempts, the message moves to the DLQ. Inspect and replay via the REST API:

```bash
# List DLQ messages
curl http://localhost:8212/event-bus/dlq

# Replay a specific message
curl -X POST http://localhost:8212/event-bus/dlq/{id}/replay
```

## Replay API

Replay any stored message range by sequence number or timestamp:

```bash
# Replay from sequence 100 to now on a subject
curl -X POST http://localhost:8212/event-bus/replay \
  -H "Content-Type: application/json" \
  -d '{"subject": "nself.cdc.np_users.*", "from_seq": 100}'
```

## REST API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | none | Health check |
| `GET` | `/event-bus/status` | admin | Broker type + subject count |
| `GET` | `/event-bus/subjects` | admin | Active subjects with message counts |
| `GET` | `/event-bus/consumers` | admin | Active consumer list |
| `POST` | `/event-bus/publish` | admin | Test-publish a message |
| `GET` | `/event-bus/dlq` | admin | Dead-letter queue listing |
| `POST` | `/event-bus/dlq/{id}/replay` | admin | Replay a DLQ message |
| `POST` | `/event-bus/replay` | admin | Bulk replay by subject + sequence range |
| `POST` | `/event-bus/purge/:subject` | admin | Purge all messages on a subject |

## Integration With CDC Plugin

When both plugins are installed, the CDC plugin publishes every WAL change to `nself.cdc.<table>.<operation>`. The warehouse plugin subscribes to `nself.cdc.*` to drive incremental exports. No manual wiring needed.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `EVENT_BUS` | `nats` | Broker: `nats` / `redpanda` / `kafka` |
| `NATS_EMBEDDED` | `true` | Start embedded NATS (only when `EVENT_BUS=nats`) |
| `NATS_URL` | — | External NATS URL when `NATS_EMBEDDED=false` |
| `REDPANDA_BROKERS` | — | `host:port,...` for Redpanda |
| `KAFKA_BROKERS` | — | `host:port,...` for Kafka |
| `KAFKA_SASL_USERNAME` | — | Kafka SASL username |
| `KAFKA_SASL_PASSWORD` | — | Kafka SASL password |
| `EVENT_BUS_RETENTION_MS` | `86400000` | JetStream stream retention (24 h) |
| `EVENT_BUS_MAX_BYTES` | `1073741824` | Max stream storage (1 GB) |

## Ports

| Port | Purpose |
|------|---------|
| 8212 | Event bus REST API (internal only, not exposed via Nginx) |
| 4222 | Embedded NATS client port (127.0.0.1 only, when `NATS_EMBEDDED=true`) |

## Docker Hub

```bash
docker pull nself/plugin-event-bus:latest
```

Image: `nself/plugin-event-bus:latest`

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_eventbus_topics` | Registered topic definitions and retention config |
| `np_eventbus_subscriptions` | Consumer registrations and delivery state |
| `np_eventbus_messages` | Message audit log (recent history; streams live in NATS JetStream) |

All tables include `source_account_id` for multi-app isolation.

## Notes

- The event bus is internal inter-plugin infrastructure. It is not a customer-facing event system and has no Hasura RLS configuration.
- Messages are durably stored in JetStream (or Redpanda/Kafka). Consumers always replay from their last acknowledged position after restart.
- `EVENT_BUS_RETENTION_MS` and `EVENT_BUS_MAX_BYTES` control how long messages are kept. Tune these based on your expected message volume.
