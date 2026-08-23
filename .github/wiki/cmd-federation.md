# nself federation

<!-- BEGIN PROSE:summary -->
> Manage GraphQL Federation for multi-service nSelf deployments (opt-in, requires `NSELF_FEDERATION=true`).
<!-- END PROSE:summary -->

## Synopsis

```
nself federation <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself federation` manages the Apollo Router supergraph that federates Hasura and plugin subgraphs into a single unified GraphQL API. Federation is disabled by default. Enable it by setting `NSELF_FEDERATION=true` in your `.env` and running `nself build`.

When federation is enabled, `nself build` collects all installed plugins with `graphql.enabled: true`, adds Hasura as the core subgraph, composes a supergraph schema via `rover supergraph compose`, and injects Apollo Router (Custom Service slot CS_7, port 4000) into `docker-compose.yml`. All `/v1/graphql` traffic routes through Apollo Router instead of directly to Hasura.

The composed supergraph schema and router configuration are written to `.nself/federation/`. The `supergraph.yaml` and `router.yaml` files in that directory are managed by nSelf and regenerated on each build.

### nself federation status
### nself federation compose
No flags.
### nself federation introspect
No flags.
## Subgraph Configuration

Plugins declare federation participation in their `plugin.yaml` manifest via a `graphql` block:

```yaml
graphql:
  enabled: true
  subgraph_name: ai          # unique lowercase name in the supergraph
  subgraph_url: http://127.0.0.1:${PORT}/graphql
  schema_path: schema.graphql  # optional static SDL for rover compose
  entities:
    - type: User
      key: id
```

`subgraph_name` must be a valid GraphQL SDL name (`[a-z][a-z0-9_]*`). `subgraph_url` may use `${PORT}`, which is interpolated from the plugin's declared port at build time. Either `schema_path` or `subgraph_url` must be non-empty for `rover supergraph compose` to succeed.

Hasura is always included as a subgraph; its URL is derived from `HASURA_PORT` in your `.env`.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `compose` | Re-compose supergraph schema from installed plugin subgraphs |
| `introspect` | Print the full supergraph schema to stdout |
| `status` | Show subgraph health and schema composition status |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Enable federation and rebuild
NSELF_FEDERATION=true nself build
```

```bash
# Re-compose after installing a plugin that exposes a subgraph
nself federation compose
```

```bash
# Check all subgraph health
nself federation status
```

```bash
# Check subgraph health in JSON format
nself federation status --json
```

```bash
# Inspect the composed supergraph schema
nself federation introspect
```

```bash
# Pipe the schema into a diff tool to detect drift
nself federation introspect > supergraph-new.graphql
diff supergraph-old.graphql supergraph-new.graphql
```

```bash
# Rebuild the stack after a composition change
nself federation compose && nself start
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [cmd-build.md](cmd-build.md) — rebuild docker-compose and nginx configs after federation changes
- [cmd-plugin.md](cmd-plugin.md) — install plugins that expose GraphQL subgraphs
- [cmd-status.md](cmd-status.md) — view the running state of your nSelf install
- [cmd-doctor.md](cmd-doctor.md) — verify environment and configuration health
- [cmd-db.md](cmd-db.md) — manage the Postgres database backing the federated schema
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
