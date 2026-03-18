# Customizing Your Stack

`nself build` regenerates `docker-compose.yml` every time. Hand-edits to that file are overwritten.

To add persistent customizations, use `docker-compose.override.yml`.

---

## How docker-compose.override.yml Works

Docker Compose automatically merges `docker-compose.override.yml` with the main `docker-compose.yml`. You only need to include the keys you want to change — everything else is inherited from the generated file.

```yaml
# docker-compose.override.yml
# This file is never overwritten by nself build.
# Use it for persistent customizations.

version: '3.8'

services:
  # Example: add extra environment variables to a plugin
  nself-ai:
    environment:
      - PLUGIN_AI_MEMORY_LIMIT=2g
      - AI_LOG_LEVEL=debug

  # Example: expose a port on all interfaces (default is 127.0.0.1 only)
  nself-mux:
    ports:
      - "0.0.0.0:3712:3712"

  # Example: mount a local config directory
  nself-claw:
    volumes:
      - ./claw-prompts:/app/prompts:ro
```

Copy the example to get started:

```bash
cp docker-compose.override.yml.example docker-compose.override.yml
```

Then edit `docker-compose.override.yml` with your changes. Run `nself restart` to apply.

---

## Persistent Resource Limits

A cleaner approach than the override file is to set env vars in your `.env`:

```bash
# .env — these survive nself build
PLUGIN_AI_MEMORY_LIMIT=2g
PLUGIN_AI_CPU_LIMIT=2.0
PLUGIN_MUX_MEMORY_LIMIT=512m
PLUGIN_DEFAULT_MEMORY_LIMIT=512m
PLUGIN_DEFAULT_CPU_LIMIT=0.5
```

These are read by `nself build` and written into the generated `docker-compose.yml` automatically. No override file needed for resource limits.

---

## What to Put Where

| Change | Recommended approach |
|--------|---------------------|
| Resource limits (memory/CPU) | `.env` vars: `PLUGIN_{NAME}_MEMORY_LIMIT` |
| Extra environment variables | `docker-compose.override.yml` |
| Extra port bindings | `docker-compose.override.yml` |
| Volume mounts | `docker-compose.override.yml` |
| Service replicas | `docker-compose.override.yml` |
| Anything else | `docker-compose.override.yml` |

---

## Notes

- `docker-compose.override.yml` is never touched by `nself build` or `nself update`.
- You do not need to include every service — only the services you want to customize.
- Run `nself restart` (not `nself build`) to apply changes from the override file alone.
- Run `nself build && nself restart` to apply both generated config and override changes.
