# Sports Plugin

> Sports scores, schedules, standings, user favorites, and multi-provider data sync. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install sports
```

## What It Does

The sports plugin pulls live and historical data from your chosen provider (theScore, ESPN, or Sportradar) and normalises it into a consistent schema covering leagues, teams, players, games, scores, and standings. Users can save favorites and receive event-driven updates when scores change.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SPORTS_PORT` | `3126` | Port the service listens on |
| `SPORTS_API_KEY` | — | API key for the chosen data provider |
| `SPORTS_PROVIDER` | — | Data provider: `theScore`, `ESPN`, or `Sportradar` |

## Ports

| Port | Purpose |
|------|---------|
| `3126` | Sports REST API |

## Database Tables

11 tables added to your Postgres database:

- `np_sports_leagues`
- `np_sports_teams`
- `np_sports_players`
- `np_sports_games`
- `np_sports_scores`
- `np_sports_standings`
- `np_sports_schedules`
- `np_sports_favorites`
- `np_sports_events`
- `np_sports_providers`
- `np_sports_sync_log`

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/sports/` | `localhost:3126` |
