# Devices Plugin

> IoT device enrollment and trust management with command dispatch and telemetry ingestion. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install devices
```

## What It Does

Handles IoT device enrollment, identity trust, and lifecycle management via a token-based registration flow. Once enrolled, devices can receive commands dispatched from the API and submit telemetry data for storage and querying. All device events — enrollment, command acknowledgement, disconnection — are recorded in a structured event log with configurable telemetry retention.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DEVICES_PORT` | `3603` | Port the Devices plugin service listens on |
| `DEVICES_ENROLLMENT_TOKEN` | — | Shared secret used during initial device enrollment |
| `DEVICES_TELEMETRY_RETENTION` | `90` | Number of days to retain telemetry data |

## Ports

| Port | Purpose |
|------|---------|
| `3603` | Devices plugin HTTP service |

## Database Tables

5 tables added to your Postgres database.

- `np_devices_registry` — Enrolled device records and metadata
- `np_devices_trust` — Device trust certificates and enrollment tokens
- `np_devices_commands` — Queued and acknowledged commands per device
- `np_devices_telemetry` — Ingested telemetry data points
- `np_devices_events` — Device lifecycle and connection event log

## Nginx Routes

| Route | Description |
|-------|-------------|
| `/devices/` | Proxied to Devices plugin service on port 3603 |
