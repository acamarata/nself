# mDNS Plugin

> Zero-config LAN service discovery via mDNS/Bonjour. **Free — MIT licensed.**

## Install

```bash
nself plugin install mdns
```

## What It Does

Advertises your nSelf services on the local network using mDNS (Bonjour/Zeroconf) so other devices discover them without manual DNS configuration. Other nSelf instances and compatible apps on the same network automatically find your services. Also stores discovered remote services in Postgres.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MDNS_PORT` | `3216` | mDNS service management port |
| `MDNS_INSTANCE_NAME` | `nself` | Service instance name on LAN |
| `MDNS_ADVERTISE_INTERVAL` | `60` | Re-announce interval in seconds |

## Ports

| Port | Purpose |
|------|---------|
| 3216 | mDNS service REST API |
| 5353/UDP | mDNS protocol (standard) |

## Database Tables

2 tables added to your Postgres database:
- `np_mdns_services` — advertised service records
- `np_mdns_discovered` — discovered remote services

## Nginx Routes

None — mDNS is a LAN protocol, not exposed via nginx.
