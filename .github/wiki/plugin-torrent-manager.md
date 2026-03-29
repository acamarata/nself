# Torrent Manager Plugin

> Torrent client with Transmission/qBittorrent compatibility and VPN enforcement. **Free — MIT licensed.**

## Install

```bash
nself plugin install torrent-manager
```

## What It Does

Manages torrent downloads via a Transmission or qBittorrent-compatible API. Enforces VPN kill-switch behavior so downloads stop if the VPN connection drops. Stores torrent state, file catalog, and download statistics in Postgres. Pairs with the vpn and content-acquisition plugins for a complete automated pipeline.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `TORRENT_MANAGER_PORT` | `3201` | Service port |
| `TORRENT_CLIENT` | `transmission` | Client backend: `transmission` or `qbittorrent` |
| `TORRENT_CLIENT_HOST` | `localhost` | Torrent client host |
| `TORRENT_CLIENT_PORT` | `9091` | Torrent client port |
| `TORRENT_DOWNLOAD_DIR` | `/downloads` | Download directory |
| `TORRENT_VPN_ENFORCE` | `true` | Kill downloads if VPN disconnects |

## Ports

| Port | Purpose |
|------|---------|
| 3201 | Torrent manager REST API |

## Database Tables

9 tables added to your Postgres database:
- `np_torrent_manager_torrents` — torrent records
- `np_torrent_manager_files` — per-torrent file catalog
- `np_torrent_manager_trackers` — tracker list
- `np_torrent_manager_categories` — download categories
- `np_torrent_manager_labels` — torrent labels
- `np_torrent_manager_stats` — download statistics
- `np_torrent_manager_rules` — auto-download rules
- `np_torrent_manager_feeds` — RSS feed sources
- `np_torrent_manager_history` — completed download history

## Nginx Routes

| Route | Target |
|-------|--------|
| `/torrent/` | Torrent manager API |
