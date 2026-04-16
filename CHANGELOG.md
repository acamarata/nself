# Changelog

All notable changes to the nSelf CLI are documented in this file. Format loosely
follows Keep a Changelog, with Conventional Commit classification.

## [1.0.8] — 2026-04-16

P92 quick-fix batch.

### Fixed

- **`nself ai` default plugin URL.** `nself ai pool list`, `nself ai local
  status`, and the rest of the `ai` subcommands defaulted to
  `http://ai:3680`, which matched neither the real plugin-ai port (3709)
  nor the generated docker-compose service name (`plugin-ai`). Users had
  to set `PLUGIN_AI_INTERNAL_URL` manually to reach the plugin from
  inside compose. Default is now `http://plugin-ai:3709`; the env-var
  override still wins for non-standard deployments.

## [1.0.7] — 2026-04-16

P92 Wave 7 companion patch release. Three bugfixes on top of v1.0.6.

### Fixed

- **Billing subcommands dispatch correctly.** `nself billing usage`, `nself
  billing invoice-preview`, `nself billing report`, `nself billing retry-event`
  now reach their handlers. The parent `billing` command was unconditionally
  returning `cmd.Help()`, shadowing every subcommand. Parent `RunE` is now
  conditional: help renders only when no subcommand was provided.
  (`cli/cmd/commands/billing.go`)

### Added

- **JWT secret auto-persist on `nself build`.** When Hasura is enabled but
  `HASURA_GRAPHQL_JWT_SECRET` is absent, the CLI generates a secure random
  secret via `crypto/rand` and writes it to `.env.secrets` with mode `0600`.
  Manual copy-from-wiki steps are no longer required for new projects.
- **`nself doctor` JWT check.** Non-zero exit code when Hasura is enabled and
  no JWT secret is found in either `.env` or `.env.secrets`.
- **Doctor test coverage.** Report aggregation, env checks, password validation
  paths now have unit tests.

### Internal

- Generated `.env.secrets` always chmod `0600` per decision D15.

## [1.0.6] — 2026-04-16

### Highlights
- P89 goreleaser configuration, homebrew release webhook fix, env variable documentation, monitoring compose wiring, compliance checker
- P90 fixes + polish across the command tree
- Coordinated release alongside plugins-pro v1.0.1 P92 (200-OK fixes, handler test coverage, AI/Google/Claw/Mux/Voice/Notify/Browser hardening, 8 Grafana dashboards, Alertmanager rules, Querier interface refactor, BaseURL injection, health tool wiring)

### Added
- P89: goreleaser config for cross-platform binaries
- P89: monitoring docker-compose wiring for Prometheus/Grafana/Loki bundle
- P89: compliance checker command
- P89: expanded env var documentation across all generators

### Fixed
- P89: homebrew formula release webhook — correct sha256 computation and URL formatting on release
- P90: assorted fixes and polish (see git log 5d232ec)

## [1.0.5] — 2026-04-15

### Highlights
- Docker DNS fix: plugin-to-plugin resolution now works via explicit compose network aliases
- Installer tarball path + filename alignment (fixes failed installs on some platforms)
- CLI error model: `os.Exit` calls replaced with returned errors across the command tree
- SSE buffer, signal handling, proxy CORS, and export pagination fixes
- 10 new `claw` subcommands for direct nClaw interaction
- Nginx resolver now uses the variable pattern everywhere, preventing Docker IP caching

### Added
- feat(claw): 10 CLI commands for interacting with nClaw (memory, topics, briefings,
  audit, tool dispatch, and pool status)

### Fixed
- fix(build): plugin compose file emits a network alias for `plugin-<name>` so
  other plugins can resolve it by short name (Docker DNS regression)
- fix: installer tarball filename and extraction path mismatch that caused some
  `curl | sh` installs to fail silently
- fix(cli): replace `os.Exit` with returned errors; correct SSE buffer flush;
  proper signal handling on shutdown; fix proxy CORS preflight; repair export
  pagination cursor math
- fix(nginx): always use the resolver variable pattern for upstreams so Docker
  container IP changes are picked up without a reload

### Changed
- P87 MEGA CLI batch: assorted stability and correctness changes rolled up from
  the ɳClaw P87 release

### Breaking
- (none)

---

Older entries are tracked in `.claude/docs/CHANGELOG.md` and in GitHub Releases.
