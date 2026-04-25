# Plugin Scope Boundaries

Every nSelf plugin owns a bounded slice of the stack. Breaking these
boundaries is how you end up with coupling, shared-database bugs, and plugins
that can't be uninstalled without collateral damage. This document defines
the contract.

## The four layers a plugin may touch

1. **Database** — own tables only. Prefix every table with `np_<name>_` (e.g.
   `np_ai_usage`, `np_mux_inbound_messages`). Foreign keys across plugin
   boundaries are forbidden; use the Hasura remote-schema layer instead.
2. **Hasura** — own roles and remote schemas. Never modify another plugin's
   permissions.
3. **HTTP** — own routes mounted under `/v1/<plugin>/...` when fronted by
   nginx. Internal ports bind to `127.0.0.1` only.
4. **Filesystem** — own subtree under `/var/lib/<plugin>/` inside the container
   and a project-scoped volume mount on the host.

## What a plugin may NOT touch

- Another plugin's database tables — ever. If you need the data, call that
  plugin's HTTP API.
- Global environment variables owned by another plugin (e.g. `AI_API_KEY` is
  owned by `ai`; `mux` must call the `ai` API over HTTP, not read the key).
- Host networking (`network_mode: host`), host filesystem paths outside the
  generated volume, or privileged Docker capabilities.
- Shared Redis keys without a namespaced prefix (`{name}:` minimum).

## Inter-plugin communication

The sanctioned channel is HTTP:

- Each plugin exposes signed HTTP using the SDK's `identity` package.
- Callers use `sdk.Identity.SignRequest(req, body)` to attach
  `X-Plugin-Signature` + `X-Plugin-Timestamp`.
- Recipients verify with `sdk.VerifyRequest(req, pubKey, body)` and reject
  requests older than five minutes.

The two well-known in-stack HTTP edges are:

| Caller → Callee | Purpose |
| --- | --- |
| `mux → ai` | Classification + summarization |
| `claw → ai` | Reasoning + embeddings |
| `claw → mux` | Email inbound / outbound |

Everything else goes through Hasura's remote schemas or Postgres row-level
views — never ad-hoc cross-plugin queries.

## Ownership rules

| Resource | Owned by |
| --- | --- |
| `np_<name>_*` tables | the named plugin |
| `/v1/<name>/*` routes | the named plugin |
| `<NAME>_*` env vars | the named plugin |
| SDK packages (`logger`, `metrics`, `server`, …) | `plugin-sdk-go` repo |
| License validation | `ping_api` service + SDK `license` package |

## Lifecycle contract

Every plugin must:

- Register with `plugin.Base` + `Info` + `Start/Ready/Shutdown`.
- Expose `/healthz`, `/readyz`, `/metrics`, `/version` via `sdk/server`.
- Emit logs via `sdk/logger` (slog JSON, `plugin=` and `version=` attrs).
- Emit metrics via `sdk/metrics` (`nself_plugin_*` names).
- Handle `SIGTERM` with a 15s drain before force-exit.

## Removal contract

Every plugin must be uninstallable by `nself plugin remove <name>` without
leaving:

- orphan tables (plugin supplies a `down.sql` migration)
- orphan Hasura roles
- orphan env vars in `.env.computed`
- orphan Docker volumes or images

Write a smoke-test in `tests/` that runs `nself plugin install` + `nself
plugin remove` and asserts the project is byte-identical to the pre-install
state.

## Enforcement

`docs/sport/F11-DEPENDENCIES.md` and the plugin-side `plugin.json` are the
machine-checked source of truth for what each plugin declares it owns. CI
fails if a plugin writes a table outside its prefix, mounts a route outside
its namespace, or calls another plugin's SQL tables.
