# nself ci

<!-- BEGIN PROSE:summary -->
> Run the nself-ci gate suite and post a GitHub commit status.
<!-- END PROSE:summary -->

## Synopsis

```
nself ci [repo-root] <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
Run the local CI gate suite for a repository.

Detects the repo stack (Go, Node/TS, Flutter) and runs the appropriate
lint, test, and build checks plus a gitleaks secret scan. Posts a
"nself-ci" GitHub commit status via gh OAuth so branch protection can
require nself-ci instead of billing-blocked GitHub Actions checks.

### Which directory is gated

`[repo-root]` is resolved against the directory you run the command from,
and defaults to `.`. Unlike the stack lifecycle commands (`start`, `stop`,
`status`, `logs`, `build`), `nself ci` is **not** redirected into a
detected `backend/` sub-directory.

That distinction matters in a monorepo. `nself start` from the repo root
deliberately retargets `backend/`, because that is where the stack lives.
`nself ci` must not: the gate belongs to the whole checkout, and the
manifests it looks for (`package.json`, `pnpm-workspace.yaml`, `go.mod`)
usually sit at the repo root while `backend/` holds only `.env`.

Before this was fixed, `nself ci --check .` in such a repo announced
"Detected monorepo layout. Using backend as project root" and gated
`backend/` instead — overriding the path you passed. If you relied on that
behaviour, pass the sub-directory explicitly: `nself ci ./backend`.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--check` | `false` | Run gates only; do not post a GitHub commit status |
| `--no-gitleaks` | `false` | Skip the gitleaks secret scan |
| `--no-status` | `false` | Alias for --check |
| `--owner` | `""` | GitHub owner (default: from git remote) |
| `--repo` | `""` | GitHub repo name (default: from git remote) |
| `--sha` | `""` | Commit SHA to report on (default: HEAD) |
| `--verbose`, `-v` | `false` | Print each gate command before running |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `eval` | Run eval-gate suite or validate an eval-set YAML |
| `forgejo` | Show Forgejo server + runner health (ops profile) |
| `serve` | Start the nself-ci webhook listener daemon (port 3845) |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
nself ci                        # gate current directory, post status
  nself ci --check                # gate only, no status posted
  nself ci --no-gitleaks .        # skip secret scan
  nself ci --sha abc1234 .        # override commit SHA
  nself ci /path/to/repo          # gate a specific repo
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
