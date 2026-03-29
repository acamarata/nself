# Home Plugin

> Home automation integration via Home Assistant REST client. Device discovery, command dispatch, state monitoring. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install home
```

## What It Does

The home plugin acts as a REST client to your existing Home Assistant instance, exposing device discovery, command dispatch, and state monitoring through your nself API. It persists device metadata and command history to Postgres so you can query and audit automations alongside the rest of your application data.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `HOME_PORT` | `3127` | Port the service listens on |
| `HOME_ASSISTANT_URL` | — | Base URL of your Home Assistant instance (e.g. `http://homeassistant.local:8123`) |
| `HOME_ASSISTANT_TOKEN` | — | Long-lived access token from Home Assistant |

## Ports

| Port | Purpose |
|------|---------|
| `3127` | Home automation REST API |

## Database Tables

3 tables added to your Postgres database:

- `np_home_devices`
- `np_home_commands`
- `np_home_states`

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/home/` | `localhost:3127` |
