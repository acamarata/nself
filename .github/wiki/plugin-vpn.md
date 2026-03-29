# VPN Plugin

> Multi-provider VPN management with kill switch and leak protection. **Free — MIT licensed.**

## Install

```bash
nself plugin install vpn
```

## What It Does

Manages VPN connections to multiple providers (NordVPN, PIA, Mullvad, and others) directly from your nSelf instance. Provides a kill-switch that blocks all traffic if the VPN drops. Monitors for DNS and IP leaks. Used alongside the torrent-manager plugin to enforce VPN on all downloads.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `VPN_PORT` | `3200` | VPN management service port |
| `VPN_PROVIDER` | — | Provider: `nordvpn`, `pia`, `mullvad`, `custom` |
| `VPN_USERNAME` | — | VPN account username |
| `VPN_PASSWORD` | — | VPN account password |
| `VPN_REGION` | `auto` | Preferred server region |
| `VPN_KILL_SWITCH` | `true` | Block traffic on VPN disconnect |

## Ports

| Port | Purpose |
|------|---------|
| 3200 | VPN management REST API |

## Database Tables

8 tables added to your Postgres database:
- `np_vpn_connections` — connection history
- `np_vpn_servers` — server catalog
- `np_vpn_providers` — provider configurations
- `np_vpn_profiles` — VPN configuration profiles
- `np_vpn_status` — current connection state
- `np_vpn_leak_tests` — leak test results
- `np_vpn_rules` — traffic routing rules
- `np_vpn_audit` — connection audit log

## Nginx Routes

None — VPN service is internal only.
