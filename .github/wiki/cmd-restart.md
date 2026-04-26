# nself restart

> Smart restart with config change detection.

## Synopsis

```
nself restart [SERVICES...] [flags]
```

## Description

`nself restart` restarts the ɳSelf stack with intelligent change detection. In the default smart mode, it compares `.env` and `docker-compose.yml` modification times against container start times. Only services whose configuration has changed are rebuilt and restarted, services with no config changes get a quick `docker compose restart` instead of a full rebuild.

For a full stop-and-start cycle (e.g., after major config changes or to clear runtime state), use `--all`. This performs a complete `nself stop` followed by `nself start`.

You can restart a single service by passing its name as a positional argument. This is useful for recovering an unhealthy container without disrupting the rest of the stack.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--smart`, `-s` | `true` | Detect changes, only restart affected services (default) |
| `--all`, `-a` | false | Full restart: stop all + start all |
| `--skip-build` | false | Skip image rebuild |
| `--no-build` | false | Alias for `--skip-build` |
| `--verbose`, `-v` | false | Detailed output |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Smart restart — only rebuilds what changed
nself restart

# Restart a single service
nself restart postgres

# Full stop + start cycle
nself restart --all

# Restart without rebuilding images
nself restart --no-build

# Restart multiple specific services
nself restart hasura auth
```

← [[Commands]] | [[Home]] →
