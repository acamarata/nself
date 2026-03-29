# Changelog

> See [CHANGELOG.md](../CHANGELOG.md) in the repo root for the full version history.

## v1.0.0

First stable release. Complete Go rewrite of the nSelf CLI (previously Bash-based v0.x).

**Highlights:**
- 24 top-level commands with 295+ subcommands
- Full Docker Compose v2 stack generation (Postgres, Hasura, Auth, Nginx)
- Plugin system with 16 free plugins and 59 Pro plugins
- `nself migrate` command for v1 → v2 migration with rollback
- Interactive `nself init` wizard with multi-env support
- `nself health watch` for continuous monitoring
- Shell completion for bash, zsh, and fish
- `nself doctor` comprehensive system diagnostics
- License management for Pro plugins
