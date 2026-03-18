# Customizing Your Stack

`nself build` regenerates `docker-compose.yml` every time it runs. Put persistent
customizations in `docker-compose.override.yml` — Docker Compose merges it automatically
on every `docker compose up`.

## Why Override Files?

Any manual change to `docker-compose.yml` is overwritten the next time you run
`nself build`. The override file is never touched by nself.

## Example: Increase Plugin Memory

```yaml
services:
  nself-ai:
    deploy:
      resources:
        limits:
          memory: 2048M
          cpus: '2.0'
```

## Example: Mount a Custom Config

```yaml
services:
  postgres:
    volumes:
      - ./custom-postgresql.conf:/etc/postgresql/postgresql.conf:ro
```

## Example: Add an Extra Environment Variable

```yaml
services:
  hasura:
    environment:
      HASURA_GRAPHQL_EXPERIMENTAL_FEATURES: "streaming_subscriptions"
```

## File Location

Place `docker-compose.override.yml` in the same directory as your `docker-compose.yml`
(your project root). Docker Compose discovers and merges it automatically — no extra
flags needed.

## Reference

- [Docker Compose merge rules](https://docs.docker.com/compose/multiple-compose-files/merge/)
- [Environment variables for resource limits](https://nself.org/docs/configuration/plugin-resources)
