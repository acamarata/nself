# Hasura Metadata Auto-Apply

`nself start` and `nself deploy` now apply `hasura/metadata/` (or the legacy `hasura/metadata.json`) automatically, if the project has one. Before this, only the manual `nself db hasura metadata apply` command ever called Hasura's metadata API — a project's on-disk metadata could silently drift arbitrarily far from what a running environment actually tracked.

## What runs, and when

| Command | When | Local or remote |
| --- | --- | --- |
| `nself start` | After health checks confirm Hasura is up (skipped with `--skip-db-init`) | Local only |
| `nself deploy local` | After the rolling restart | Local |
| `nself deploy staging` / `nself deploy prod` (host configured) | After the remote rolling restart, over SSH | Remote — `hasura/` is rsynced alongside the compose file and env, then the remote host's own `nself db hasura metadata apply` is invoked |
| `nself deploy staging` / `nself deploy prod` (no host configured) | After the local rolling restart (assumes the deploy is being run from the target box itself) | Local |

Every path is a clean no-op when the project has no `hasura/metadata/` directory and no `hasura/metadata.json` — and, separately, when nothing responds on Hasura's configured port at all (a stack/run that doesn't include Hasura). The second check only distinguishes "no Hasura here" (connection refused/timeout) from "Hasura is here but unhealthy" (any HTTP response, including a degraded/down `/healthz`) — the latter still goes through the normal strict/warn-only apply attempt below, since a real instance may just be slow to come up.

## Strict vs. warn-only

Controlled by `NSELF_HASURA_METADATA_STRICT` (`true`/`false`). If unset, the default is:

- **dev** — warn-only. A failed apply, or metadata that applies but leaves inconsistent objects (`get_inconsistent_metadata`), prints a warning and does not fail the command.
- **staging / prod** — strict. The same failures fail `nself start` / `nself deploy`.

Override either direction explicitly:

```bash
# Fail dev builds on metadata problems too (e.g. CI):
NSELF_HASURA_METADATA_STRICT=true nself start

# Downgrade a prod deploy to warn-only while investigating a known-bad metadata tree:
NSELF_HASURA_METADATA_STRICT=false nself deploy prod
```

## Manual application

```bash
nself db hasura metadata apply     # apply hasura/metadata/ by hand
nself db hasura metadata export    # export live metadata to git-friendly sorted YAML
nself db hasura diff               # compare live metadata against on-disk files
```

`nself db hasura metadata apply --env staging` (or `--env prod`) re-invokes the same command on that environment's own remote host over SSH — see `nself db hasura --help`. (A dedicated top-level `nself hasura` alias was considered but dropped: CLI-R11 caps the core command surface, and `nself db hasura ...` already covers this — see `.github/command-surface-budget.txt`.)

## Malformed `!include`'d table files

Hasura CLI-style metadata (`hasura/metadata/tables.yaml` + one `!include`'d YAML file per table) requires each included file to be a bare mapping (`table: {...}`), not a one-element list (`- table: {...}`). A list-wrapped file is a common copy-paste mistake when hand-editing metadata and previously reached the Hasura API as silently malformed metadata. `nself db hasura metadata apply` now catches this before sending anything, with an error naming the exact section and index, e.g.:

```
hasura metadata: tables[12] is a YAML list, not an object — the !include'd file is
list-wrapped (starts with "- " instead of a bare mapping); unwrap it to a single object
```

## Related pages

- [[operations/hasura-metadata-backup]] — daily export + restore
- [[cmd-start]] — full start command reference
- [[cmd-deploy]] — full deploy command reference
- [[Home]]
