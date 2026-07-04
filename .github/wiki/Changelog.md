# Changelog

All notable changes to the ɳSelf CLI are documented in this file. Format loosely
follows Keep a Changelog, with Conventional Commit classification.

## [1.2.4] — 2026-07-03

### Fixed

- **License signature verification enabled in released binaries (CRITICAL)** — the release workflow read the `NSELF_LICENSE_PUBKEY_HEX` secret to embed the Ed25519 license-verification key at build time, but the secret was never set in the repository, so every published binary shipped as a "dev build" with license signature verification disabled and printed a dev-build warning from `nself version`. The secret is now configured; binaries from this release on verify license signatures.
- **`bundle info sentry` alias miss** — the `sentry` → `nsentry` alias added in 1.2.3 resolved for `bundle install` and `bundle remove` but not `bundle info`, which used a separate command-layer lookup. `bundle info sentry` now resolves to ɳSentry; listings and error hints still show only the canonical `nsentry` slug.
- **gofmt drift** — three files merged unformatted; formatted so the CI gofmt gate stays green.

## [1.2.3] — 2026-07-03

### Added

- **`sentry` bundle install alias** — `nself bundle install sentry` now resolves to the canonical `ɳSentry` bundle. The alias is a fallback lookup only; `nself bundle list` and shell completions continue to show the canonical `nsentry` slug. (#171)

## [1.2.2] — 2026-07-03

### Fixed

- **Declared plugins silently dropped (CRITICAL)** — `nself build` ignored the project's `nself.yaml` manifest entirely: plugins declared under `plugins:` (flat or `free:`/`pro:` tiers) or via `bundle:`/`bundles:` never materialized as containers because the only injection path was discovery over the plugin install dir. The build now parses the manifest, expands bundles through the canonical bundle catalog, best-effort auto-installs missing plugins (60s per-plugin timeout; disable with `NSELF_AUTO_INSTALL_PLUGINS=false`), and reports any plugin it cannot wire with a per-plugin warning, a `BuildResult.MissingPlugins` entry, and a printed `nself plugin install ...` fix command. A declared plugin is never silently dropped. Core services (`auth`, `storage`) satisfy their declared names. `NSELF_PLUGIN_DIR` overrides the plugin directory for hermetic CI and per-project plugin sets. (#168)
- **Literal secrets in generated docker-compose.yml (HIGH)** — `nself build` baked literal secret values (Postgres password, Hasura admin secret, JWT blobs, MinIO root credentials, SMTP password) into the generated compose file, so editing `.env` post-build was a silent no-op and committing the generated file leaked credentials. Secrets are now emitted as `${VAR}` references; interpolation values are written to `.nself/compose.env` (mode 0600) and passed to docker compose via `--env-file` by start/stop/restart/health. Legacy projects without `compose.env` keep default `.env` discovery. A safety-net scan warns if any known secret value still appears literally. (#168)
- **False "unknown env var" warnings** — app-owned vars (`NODE_ENV`, `JWT_SECRET`, `SSL_AUTO_TRUST`, `COOKIE_SECRET`, `ENABLE_DEBUG`, `LOG_LEVEL`, `NSELF_PROJECT_NAME`) are now known; new `ENV_ALLOWLIST` accepts comma-separated exact names or `PREFIX_*` patterns for project-specific passthrough vars. (#168)
- **Unreachable ready URLs from `nself start`** — with a default local domain, `nself start` printed `*.local.nself.org` URLs that 502 without DNS setup. It now prints direct `http://localhost:<port>` endpoints (GraphQL, Hasura console, Auth, Storage, Mail UI, Admin) plus a `nself dns-setup` hint for the domain routes; custom domains keep nginx-routed URLs. (#168)
- **Hash-prefixed leftover containers** — interrupted `docker compose` recreates leave the old container renamed with a hex-id prefix (e.g. `b6d7b59a1c78_myapp_hasura`) that holds ports and shadows the clean `<project>_<service>` name. Start now removes rename-leftovers before and after `compose up`. (#168)

## [1.2.1] — 2026-07-03

### Fixed

- **Migration ledger PK collision (Unity HIGH PCI)** — nested Hasura layouts (`hasura/migrations/default/<name>/up.sql`) keyed the applied-migration ledger by `filepath.Base(file)`, collapsing every nested migration to the literal name `up.sql`. The first one recorded; every later one was silently skipped while reporting as applied (this produced the ummat `is_up`/`np_uptime_summary` ghost). Migration identity now derives from `migrationKey()` (parent directory name for nested layouts, filename for flat), `extractMigrationID` no longer truncates at the first `_` (same-day migrations no longer collide), ledger writes use `ON CONFLICT DO UPDATE` instead of `DO NOTHING` (a conflict can never silently drop a migration), and `upgradeLedger()` rewrites legacy ledger rows in place on already-deployed boxes. (#163)
- **Atomic dual-table ledger recording** — non-transactional migrations (`CREATE INDEX CONCURRENTLY`, `ALTER TYPE ADD VALUE`) now record to `np_common.schema_versions` and `nself_ops.migrations` inside one transaction, so a failure on either INSERT leaves no orphan row. `nself db migrate down` deletes from BOTH ledgers inside the down transaction (previously `nself_ops.migrations` kept a stale row). `nself db migrate apply` errors on checksum mismatch when a previously-applied file was modified. (#141)
- **Scaffold template error propagation** — plugin scaffold template failures now return wrapped errors instead of panicking; generated service templates exit cleanly instead of `os.Exit(1)` mid-defer. (#141)
- **clean-root SDK gate** — expects 3 SDK modules (go, py, ts) after the Flutter SDK removal (#159); Windows CI perm-bit assertions POSIX-gated. (#164)
- **Version scripts** — `bump-version.sh` / `check-version-lockstep.sh` no longer expect the removed `sdk/flutter/pubspec.yaml` (10 lockstep files + optional homebrew).

### Added

- **`nself db migrate status --migration-dir <dir>`** — status for non-standard migration layouts (parity with `migrate up`; forwarded on `--env` remote dispatch). Repos like ntask (`postgres/migrations`) no longer report "No migrations found". (#166)
- **`postgres/migrations/` auto-detect** — 4th candidate in migration directory detection. (#166)
- **Remote CLI version-drift pre-flight** — `db --env staging|prod` commands verify the remote binary version matches local before running, failing with a clear message naming both versions (pass `--allow-version-drift` to override). (#162)
- **`.github/wiki/database-migrations.md`** — internal reference for the dual-table ledger, atomicity guarantees, and layout auto-detection. (#141, #166)

## [1.2.0] — 2026-07-01

### Changed

- **TypeScript SDK canary upgrade to TS6** — `cli/sdk/ts` (@nself/plugin-sdk v2.0.0) bumped from TypeScript `^5.4.5` to `^6.0.0` (S06 canary). `tsconfig.json` updated: `module` → `node16`, `moduleResolution` → `node16` (node10 alias deprecated in TS6). Added `tsconfig.test.json` for ts-jest with `isolatedModules: true`. Jest config migrated to flat `transform` syntax (removes deprecated `globals.ts-jest`). Added `eslint`, `@typescript-eslint/parser`, `@typescript-eslint/eslint-plugin` devDeps and `eslint.config.mjs` (ESLint 10 flat config). All 18 tests pass. Type-check and build clean.

### Added

- **Embedded PostgreSQL via pglite/wasmtime** (`--embedded-pg`) — `nself start --embedded-pg` boots a full ɳSelf stack without a Docker postgres container. PostgreSQL runs as a pglite WebAssembly module inside wasmtime. A Unix-domain socket bridge proxies the Postgres wire protocol to Hasura and Auth. pgvector is included in pglite v0.2.17. Cold start compiles the WASM module to native code and caches it; warm starts typically take under 5 seconds. Backup via `nself backup` targets the UDS socket (`pg_dump --format=custom` primary, SQL wire-protocol fallback). Enable persistently with `NSELF_EMBEDDED_PG=true` in `.env.local`. See [[Embedded-Postgres]].

### Security

- **CWE-214 Hasura secret exposure fixed** — `hasuraMetadataExportCmd()` in `internal/backup/create.go` previously passed the Hasura admin secret as `--admin-secret=<value>` in argv. Any local user with access to `/proc/<pid>/cmdline` or `ps aux` could read it during a backup run. The secret is now passed exclusively through the child process environment (`cmd.Env`). Docker exec receives `-e HASURA_GRAPHQL_ADMIN_SECRET` (no value on the command line). Severity: High. Chain ID: a83c99d6. Advisory: `.github/SECURITY-ADVISORIES/2026-05-15-rce-and-secrets.md`.
- **Supply-chain installer verification added** — the Ollama installer previously piped `curl` output directly to `sh` without content verification. `DownloadAndVerify()` in the new `internal/installer/verify.go` downloads to a 0700 owner-only temp directory, opens the file with `O_EXCL` (closes TOCTOU window), caps the body at 2 MiB, and verifies SHA-256 against the pinned checksum from `ExpectedOllamaInstallChecksum()` before execution. Severity: High. Chain ID: a83c99d6. Advisory: `.github/SECURITY-ADVISORIES/2026-05-15-rce-and-secrets.md`.

### Testing

- **S42-sec security-critical coverage uplift** (sprint a5e2b723) — targeted coverage pass across `internal/license`, `internal/auth`, `internal/trust`, `internal/crypto`, SSRF guards, and RLS enforcement. Real uplift: license 76 → 91.1% (392 new tests), auth 85 → 92.5% (140 tests), trust 67 → 72.1% (4 timeout-panic test bugs fixed: brew/osascript/setupResolver had unconditional 30s–5 min blocking calls, now `t.Skip` with unreachable-doc comment), security 89.3%, truststate 96.0%. No security bugs discovered; `go build` and `go vet` clean. Accepted coverage debt: `internal/trust/ssl` 49.4%, `internal/secrets` 39.0%, `internal/tenant` 48.4% — these packages rely on external binary execution (mkcert, openssl, age — no injection point to test safely), osascript admin-dialog invocation (untestable in CI without real macOS privileges), and live Postgres/Stripe/MinIO connections. Plausible integration-test class; further uplift deferred to S47 once an integration harness is in place.

## [1.1.3] - 2026-05-15

### Docs

- **Complete CLI wiki coverage** — 19 new command pages authored (S05): `cmd-ai-studio.md`, `cmd-costs.md`, `cmd-encryption.md`, `cmd-feature.md`, `cmd-federation.md`, `cmd-help-topics.md`, `cmd-infra.md`, `cmd-k8s.md`, `cmd-man.md`, `cmd-mcp.md`, `cmd-migrate-from-v099.md`, `cmd-ollama.md`, `cmd-region.md`, `cmd-release.md`, `cmd-release-check.md`, `cmd-release-rollback.md`, `cmd-release-status.md`, `cmd-self-heal.md`, `cmd-uninstall.md`. Wiki coverage advances from 66 to 85 pages.

## [1.1.2] - 2026-05-15

Patch release. P101 nClaw groundwork: nself-sync server, nself-vault KEK envelope, LlamaCpp real backend, sqlite-vec cross-compile matrix, throttle retries, nself-audit baseline rules. Security hardening across signing, vault revocation, license HMAC, and Argon2id KAT. Doc-truth corrections to SPORT (F01/F02/F04/F09) and PPI plugin counts.

### Added

- **nself-sync server** — push, subscribe, ack, and snapshot handlers wired end-to-end.
- **nself-vault KEK envelope encryption** — root-key wrapping with documented rotation procedure.
- **LlamaCpp real backend** — GPU offload, sampling, streaming, and memory guards.
- **sqlite-vec cross-compile CI matrix** — 5 target combinations covered.
- **Throttle retries with full jitter** — honors `Retry-After` headers when present.
- **nself-audit baseline rules** — 10 baseline scan rules integrated into `nself doctor --deep`.
- **@nself/config workspace package** — scaffold for shared configuration.
- **F09 ENV-VAR-INVENTORY** — 992-line catalog covering v1.2.0 forward-looking vars.

### Fixed

- **`/healthz` nginx proxy_method set to GET** — `internal/nginx/templates/service.conf.tmpl` lines 43-52 previously used `proxy_method POST` for the upstream health check location. Some plugin runtimes return 405 on a POST to `/healthz`. Changed to `proxy_method GET` so `nself doctor` health probes reliably succeed across all plugin types.
- **Cross-language signing material** — Rust and Go produce byte-identical signing bytes. 119-byte golden test locked.
- **nself-vault REVOKE** now invalidates immediately. JWT `aud="nself-vault"` enforced. Cross-ownership reads return 404 (not 403).
- **Plugin signing** uses canonical SHA-256 of tarball bytes. Worker and CLI aligned.
- **License HMAC key** randomized at provisioning. No longer derived from an observable value.
- **Argon2id KAT test mismatch** — test was wrong, production `derive_key` was always correct.
- **Tauri 2 updater chain** — plugin declared in Cargo.toml, Ed25519 minisign signing, real public key, downgrade_guard.
- **nclaw/desktop Tauri 2 API drift** — 7 compile errors cleared.
- **nclaw/core test surface** — 16 compile errors plus 15 surfaced runtime failures fixed.
- **WebSocket goroutine leak** — no fd exhaustion on aggressive context cancellation.
- **TODO / stub / unimplemented! markers** — removed from all production paths.

### Security

- All TLS, WAF, and hardening rules ship free at install, update, deploy, and daily scan (Security-Always-Free).
- AGPL/SSPL gate active in fail mode across cli, admin, plugins, plugins-pro, web.
- `nself doctor --deep` runs without a license. Critical findings exit 1.

### Changed

- SPORT F01 / F02 / F04 / F09 regenerated against code reality.
- PPI corrections: 87 → 112 paid plugins. 25 → 29 free plugins.
- ɳ branding enforced across user-visible prose for products, bundles, pricing.

### Docs

- 11 CLI wiki `cmd-*.md` pages promoted from v1.0.9 PREVIEW to v1.1.1 SHIPPED status.
- README versions bumped (cli, admin, clawde).
- Tauri updater signing procedure documented.
- KEK rotation procedure documented.
- Mobile platform encryption matrix published — iOS, Android, macOS encrypted; Linux, Windows, web unsupported.
- ADR-003 records admin Next.js permanent exception.

### Known limitations (carry-forward to v1.1.3)

- Integration test API drift: httpmock 0.7 → 0.8, nclaw_core → libnclaw rename. Separate sprint.
- 3 CLI commands still need dedicated wiki pages (model, template, migrate firebase/supabase variants — addressed in S28 SPORT regen).
- Throttle retry orchestrator integration deferred to S17.T07.

## [1.1.0] - 2026-05-15

Minor release. ɳSentry bundle (13 plugins), ClawDE bundle buyable, ɳFamily ratified, nCloud waitlist mode. Observability auto-wiring (Prometheus scrape, Loki/Promtail, Grafana dashboards), backup drill, env migration tooling, idempotent admin trust install.

### Added

- **`nself bundle install <name>`** (S13.T11) — install all plugins in a bundle in one command. Supported: `sentry` (13 plugins), `family` (9 plugins), `clawde` (8 plugins), `claw`, `chat`, `tv`, `task`. Requires bundle or ɳSelf+ entitlement.
- **`nself bundle remove <name>`** (S13.T11) — uninstall every plugin in a bundle, reverse dependency order.
- **`nself bundle list`** (S13.T11) — show all 7 bundles (6 paid + ɳTask free) with install state, plugin counts, license tier.
- **`nself bundle info <name>`** (S13.T11) — print bundle membership, plugin versions, ports, entitlement requirements.
- **`nself feature list`** (S13.T12) — list all feature flags (cloud-waitlist, sentry-rum-cdn, family-csam-strict, etc.) with current state.
- **`nself feature enable <flag>`** (S13.T12) — flip a feature flag on at runtime; persisted in `.env.features`.
- **`nself feature disable <flag>`** (S13.T12) — flip a feature flag off.
- **`nself feature status <flag>`** (S13.T12) — show one flag's state plus the source (env, file, default).
- **`nself backup drill`** (S13.T13) — run the full backup → restore → verify cycle against a scratch DB; reports RTO/RPO measured timings. Wired into `OPS-DRILL-01` doctor check.
- **`nself man`** (S13.T14) — generate man pages from cobra command tree; installs to `$prefix/share/man/man1/nself*.1`.
- **`nself costs`** (S13.T15) — estimate monthly infrastructure cost (Hetzner sizing × VPS class × plugin storage); reads `costs.yaml` plugin annotations.
- **`nself migrate firebase`** (S13.T16) — assisted import from Firebase: Auth users → nHost Auth, Firestore → Postgres + Hasura, Storage → MinIO. Dry-run by default; `--apply` to commit.
- **`nself migrate supabase`** (S13.T16) — assisted import from Supabase: pg_dump → restore, Storage → MinIO, Edge Functions → nself Functions.
- **`nself sentry status`** (S13.T11) — surface ɳSentry health (uptime, incidents, SLOs, alerts) at a glance.
- **`nself cloud provision`** (S12.T07) — stub provisioning command for nCloud managed hosting; returns waitlist enrollment response.
- **`nself cloud status`** (S12.T07) — check provisioning and plan status for nCloud-managed instances.
- **`nself family status`** (S11.T04) — show ɳFamily plugin status and CSAM scan health.
- **`nself tenant create`** / **`nself tenant list`** (S12.T08) — Cloud multi-tenancy tenant record management (`tenant_id` UUID per Convention Wall).
- **13 new CLI commands for ɳSentry plugins** (S10.T01..T13): `sentry uptime`, `sentry status-page`, `sentry incident`, `sentry alert-router`, `sentry slo`, `sentry synthetic`, `sentry rum`, `sentry errors`, `sentry cron-monitor`, `sentry oncall`, `sentry crash`, `sentry anomaly`, `sentry audit`.
- **ɳSentry Prometheus auto-scrape** (S10.T16) — `nself build` emits scrape_configs targeting every installed ɳSentry plugin endpoint; no manual prometheus.yml edits.
- **Loki + Promtail build wiring** (S10.T17) — `nself build` provisions Loki on port 3100 and Promtail tail rules for plugin containers; structured log ingest by default.
- **ɳSentry Grafana dashboards** (S10.T18) — 13 pre-built dashboards (uptime, incidents, SLO burn, RUM CWV, anomaly) auto-imported on `nself start` when Grafana is enabled.
- **Alertmanager nsentry receiver** (S10.T19) — alert routing config block generated when ɳSentry bundle is installed; routes critical alerts to alert-router plugin.
- **Doctor check `OBS-SCRAPE-01`** (S10.T16) — verifies every ɳSentry plugin endpoint is scraped by Prometheus.
- **Doctor check `OPS-DRILL-01`** (S13.T13) — verifies backup drill has run in the last 7 days; warns at 14d, fails at 30d.
- **Doctor check `OBS-REDACT-01`** (S10.T20) — verifies log/metric redaction rules are present in Promtail config for PII fields.
- **Doctor check `LEGAL-COPPA-01`** (S11.T08) — verifies COPPA age-gate is enabled when ɳFamily social plugin is installed.
- **Doctor check `LEGAL-GDPR-A9-01`** (S11.T09) — verifies GDPR Article 9 special-category-data consent flow is wired when family medical plugins are installed.

### Changed

- **License gate** (S08.T03) — `nself plugin install` now checks ɳSentry bundle entitlements for all 13 ɳSentry plugins.
- **`nself doctor`** (S10.T16, S13.T13, S10.T20, S11.T08, S11.T09) — five new checks added (OBS-SCRAPE-01, OPS-DRILL-01, OBS-REDACT-01, LEGAL-COPPA-01, LEGAL-GDPR-A9-01).
- **Minimum nSelf CLI version requirement** for ɳSentry, ɳFamily, nCloud features: v1.1.0.
- **Brand display** updated in command help text — ɳSelf eta marks now render in non-ASCII-stripped help (S13.T22).

### Fixed

- **Idempotent macOS trust install** (S13.T05) — `nself trust install`, `nself dns-setup`, `nself ports`, `nself ssl install` now state-check before invoking `osascript with administrator privileges`. Eliminates the 24-prompt burst incident (Admin Prompt Hygiene Hard Rule). Calls return immediately when target state is already configured.
- Port collision resolution (S13.T06): ports 3820–3849 block fully documented and enforced in `nself doctor --ports`.
- `nself build` no longer emits stale `prometheus.yml` blocks when bundles are removed (S10.T16).

### Deprecated

- **Legacy `nself monitor` subcommands** (S10.T21) — `nself monitor uptime` and `nself monitor status` are superseded by `nself sentry uptime` / `nself sentry status-page`. Wrappers remain for one minor cycle; will be removed in v1.2.0.

### Security

- Trust install state-checks (S13.T05) close the burst-prompt vector where 30 parallel agents could stack 24 macOS auth dialogs in <30s — see Admin Prompt Hygiene Hard Rule in PPI.
- Log redaction (OBS-REDACT-01, S10.T20) ensures PII fields (email, phone, full-name) are redacted at ingest time, never persisted in Loki.

---

## [Unreleased] — v1.0.14

P98 Batch 1. Performance hardening and operational documentation.

### Added

- **Redis connection-pool tuning** (P98-T01). `REDIS_POOL_SIZE`, `REDIS_MIN_IDLE`, `REDIS_CONNECT_TIMEOUT_MS`, `REDIS_READ_TIMEOUT_MS`, `REDIS_WRITE_TIMEOUT_MS` env vars. Pool defaults to `runtime.NumCPU() * 2` with a min-idle of 2. Backoff on failed pool acquisition. Docs: [[operations/redis-tuning]].
- **MeiliSearch index warm-up** (P98-T02). `MEILISEARCH_WARMUP_ENABLED` + `MEILISEARCH_WARMUP_INDEXES` env vars. Warm-up runs on `nself start` after service health check passes; re-runs on config change detected by the watchdog. Docs: [[operations/meilisearch-warmup]].
- **JWT key rotation operations page** (P98-T03). Documents the zero-downtime dual-key rotation flow (already shipped v1.0.10). Includes env var reference, rotation runbook, and rollback steps. Docs: [[operations/jwt-rotation]].
- **docker-compose.yml header audit** (P98-T05). 108 generated compose files across the ecosystem now carry the `# GENERATED BY nself build — DO NOT HAND EDIT` header. nSelf-First Doctrine CI gate enforces this on every PR.
- **SPORT F02 sync — pentest-kit** (P98-T06). `nself pentest-kit` added to the command inventory (F02-COMMAND-INVENTORY.md). Command count: 83.
- **Bus-factor D9 backup-admin deferrals** (P98-T07). D9 deferred for 9 external accounts (Apple Developer, Google Play, LiveKit, HubSpot, Email-on-Acid, GitHub Sponsors). Documented in `bus-factor.md` with deferred-until date and re-evaluation trigger.

### Notes

- No new CLI commands added to the binary in this batch (pentest-kit existed; F02 was stale).
- No version bump yet. v1.0.14 tag pending user approval.

### Added (Batch 2)

- **Hasura metadata backup cron** (P98-T13). Daily 02:00 UTC backup via `cli/internal/backup/hasura_metadata.go` and `cli/internal/maintenance/hasura_metadata_cron.go`. Systemd timer + macOS LaunchDaemon (TZ=UTC enforced). New `BACKUP-METADATA-01` doctor check in `--deep`. File mode 0600. Docs: [[operations/hasura-metadata-backup]].
- **SSRF guard partial — claw DNS-rebinding hotfix** (P98-T12 partial). Closes a TOCTOU bug in claw browser client. Multi-service migration to a unified shared SSRF package (notify, mux, browser, ai) deferred to v1.1.0 per Opus CR-C findings.
- **JWT key rotation hardening** (P98-T11 fixes from CR-C). 11 follow-on fixes from the security review: `flock(2)` on rotation log to prevent concurrent races, XDG_STATE_HOME fallback for log path, `--to-file` and `--no-print` flags on `nself self-heal --jwt`, escalate-to-fail in JWT-ROT-01 doctor check, tighter dir perms (0700), strconv.Atoi for env parsing. 14 new tests covering concurrency, crypto round-trip, dry-run, error paths.
- **Multi-tenant convention wall — web docs** (P98-T08). `web/docs/src/content/multi-tenancy/conventions.mdx` documents the `source_account_id` (multi-app) vs `tenant_id` (Cloud) distinction with a decision tree. Companion to the `PERM-RLS-01` doctor check.
- **AGPL/SSPL warn-gate uniform across 5 repos** (P98-T04). Workflows standardized in cli, plugins (license-gate.yml), plugins-pro, admin (license-gate.yml), web. All warn-only through 2026-05-20 triage window, then flips to fail-PR.
- **Bus-factor D9 deferrals** (P98-T05). 9 critical vendor accounts marked DEFERRED to P99 per the D9 escape hatch, awaiting user backup-admin nominations.
- **Secondary-domain Namecheap verification** (P98-T07). clawde.io / clawde.net / claw-de.com confirmed registered at Namecheap (expiry 2027-02-16). Transfer-lock OFF flagged to user as T1-28.
- **CLI gap catalog T1 mappings** (P98-T02). G-001..G-008 in `nself-first-cli-gaps.md` now have explicit T1 user-decision blocks (T1-23..T1-26).

### Changed (Batch 2)

- **ntask now nSelf-First** (P98-T14). The `ntask/` reference app no longer uses `docker-compose up` directly. `make up` and `make down` delegate to `nself start` / `nself stop`. The D6 "any-stack" exception is superseded.
- **Compose audit doc reconciled** (P98-T01 follow-up). The 130-file ecosystem inventory at `.claude/docs/doctrines/nself-first-compose-audit.md` had per-category counts corrected.

### Security (Batch 2)

- **claw DNS-rebinding TOCTOU closed** (P98-T12 hotfix). The claw browser http.Client now uses a Transport with DialContext that re-validates resolved IPs at dial time, blocking RFC1918, link-local, loopback, and metadata IPs.
- **Doctor SSRF-01 honesty fix**. The check no longer passes vacuously on file-stat alone. It now verifies guard packages reference `DialContext` and `IsBlockedIP`-style guard symbols. Three states: PASS, WARN, FAIL.
- **Secret-scrub runbook published**. `.claude/docs/operations/secret-scrub-runbook.md` documents triage, rotation, and (when authorized) git-history scrub procedures. Cross-references bus-factor and destructive-deny-list rules.

### Notes (Batch 2)

- 02.T11 CRIT-1 (JWT dual-key grace period not implemented in code despite documentation) is escalated to T1-27. User must choose: implement real JWKS dual-key support (defer to v1.1.0) or strip grace-period language from code and docs (XS effort, ship-ready).
- 02.T12 multi-service SSRF migration captured in `.claude/ideas/p99-ssrf-shared-migration.md` for v1.1.0.
- 8 qa/bugs closed by the STORM rigor pass on 2026-04-30: BUG-16dd1758, BUG-52c481a1, Chain-fcc4ef6e, chain-50e9faf5, Chain-48771a51, admin-lockstep-drift, og-package-untracked, trivy-action-kev-cve.

---

## [Unreleased] — v1.0.13

P97 Wave 11. CLI coverage gates extended past the 75% per-package floor.

### Changed

- **Coverage gate (`.github/workflows/coverage.yml`) extended** to enforce 75% per-package floor on `internal/trust`, `internal/ui`, `internal/watchdog` alongside `internal/auth` + `internal/license` (G0-T11). Path A fix per CI/CD 100% Green Hard Rule: root-cause coverage authoring, not gate lowering.
- **`internal/trust` coverage 20% → 76.2%**. Adds testability seams: `currentOS()` drives the cross-platform switch; `findDnsmasqConfFunc` redirects `configureDnsmasqConf` at a temp path; `setup{DNSDarwin,Mkcert,PortsDarwin,DNSLinux,PortsLinux}Func` drives `setupDarwin` / `setupLinux` success and error branches without admin prompts. Platform guards via `t.Skip` only (G0-T11).
- **`internal/ui` coverage 10% → 97.5%**. Adds `stdoutIsTerminalFunc` to drive TTY-only goroutine paths in `Spinner.Start`, `FirstRunProgress`, `DockerPullProgress`, `ProgressBar.render` (G0-T11).
- **`internal/watchdog` coverage 51% → 94.3%** (G0-T11).
- **`Contributing.md` documents the new per-package coverage floors** (G0-T11).

### Notes

- No skip mechanisms added (no `continue-on-error`, no `.skip()`).
- No production behavior change. Refactors are testability seams only.

---

## [1.0.10] - 2026-04-22

SP-01 Wave-0+1 patch release. Version bump, ping_api env var update, and plugin
tarball pipeline fix. No new CLI commands or breaking flag changes.

### Fixed

- **CLI version bump v1.0.9 to v1.0.10** (SP-01.R01-T01). `nself version` now
 reports `1.0.10`.
- **ping_api `LATEST_CLI_VERSION` env var updated to `1.0.10`** (SP-01.R01-T03).
 `nself update --check` and `nself doctor` resolve against the correct version.
- **plugins-pro tarball release pipeline** repaired (SP-01.Y32). Switched CI
 runner from `macos-latest` to `ubuntu-latest`; `build-tarballs.sh` now
 produces correct release artifacts on every tag push.

---

## [1.0.9] - 2026-04-18

P93 long-term support release. 50-sprint phase covering CLI stabilization, admin parity, plugin
depth work, reference app polish, web monorepo refresh, release engineering, and
full doc-sync ritual. CLI and admin now ship in lockstep (same version, same cadence).

### Added

- **CLI = Admin lockstep versioning.** From v1.0.9 onward, the CLI binary and the
 `nself/nself-admin` Docker image carry the same version number. Version bumps
 go through a single coordinated release. Both surfaces can drive local,
 staging, and prod environments from a single install (S05, S27).
- **`nself license` subcommand family hardened.** `license set`, `license show`,
 `license revoke`, `license simulate`, and `license grace` round out the
 validation flow against `ping.nself.org`. Grace window, revocation handling,
 and offline-mode (`NSELF_LICENSE_SKIP_VERIFY=1`) all documented in
 `docs/LICENSING_SPEC.md` (S43).
- **`nself doctor --deep` free-tier hardening sweep.** Runs the full Security-
 Always-Free audit: RLS coverage, rate-limit registry, MFA throttle, SSRF guard,
 JWT key age, WAF baseline, audit-log config, encryption-at-rest, SIEGE
 regression suite, automatic TLS. No license check. Fires on every
 `install / update / deploy` + daily cron (S10, S44).
- **ɳSelf-First Doctrine CI gate.** `.github/workflows/nself-first-check.yml`
 fails CI on any `docker-compose up` or `docker compose up` outside
 `nself build` / `nself start` in `task/`, `chat/`, `claw/`, `clawde/`,
 `ntv/`, `web/backend/`. Doctrine + audit + gap catalog in
 `.claude/docs/doctrines/` (S41).
- **Doc-sync ritual enforcement.** `nself --help`, flag strings, env var names,
 and SPORT file entries cross-verified in CI. Every PR touching a user-visible
 surface requires matching wiki + docs changes per the `pre-commit.md`
 change-type matrix (S42).

### Fixed (P94 patch fixes)

- **Plugin install now accepts multiple plugin names.** `nself plugin install ai claw mux`
 installs all three in dependency order. A failure on one plugin does not abort the others;
 the final exit code reflects any partial failure (S01.T01).
- **Plugin conflict error now names both competing plugins.** Route conflicts report
 `route conflict: /api claimed by <pluginA> and <pluginB>` instead of a generic
 server_name message (S01.T02).
- **Plugin install no longer false-positive conflicts with base service routes.**
 On a clean `nself init`, installing any plugin no longer reports phantom conflicts on
 `/api` (Hasura), `/auth`, or `/storage`. Plugins that genuinely claim those paths
 still error with the owning base service named (S01.T03).

### Fixed

- **Plugin registry drift.** SPORT F04 pinned at 109 pro plugins (filesystem +
 registry + hardcoded list all reconciled). No more 59 vs 63 discrepancies (S09).
- **Master-lists / FEATURES.md zero-yellow gate.** Every feature with a Partial
 status reconciled to Done or explicitly downgraded with user approval (S06, S12).
- **Environment cascade precedence.** `.env.secrets` and `.env.computed` now load
 in the documented order even when `.env.local` overrides a subset of keys (S04).
- **Admin to CLI parity gaps closed.** License management, plugin install/remove,
 environment switching, and multi-env deploy all reachable from both surfaces
 with identical payloads (S05, S27).

### Changed

- **`docker-compose.yml` header mandated.** Every generated compose file carries
 `# GENERATED BY nself build -- DO NOT HAND EDIT`. Hand-edited files fail the
 ɳSelf-First CI check (S41).
- **License key grandfathering.** Existing $9.99/yr keys map to Basic tier; all
 new keys follow the 7-tier matrix in `F07-PRICING-TIERS.md` (S43).
- **CLI help output pulls from SPORT.** Command counts, plugin counts, bundle
 membership all sourced from `.claude/docs/sport/` at help-render time (S42, S48).

### Security

- **Security-Always-Free baseline.** All core hardening features ship free and
 fire automatically on every install/update/deploy + daily. Pro/Cloud tiers
 layer on analytics, not baseline controls (S10, S44).
- **JWT key rotation default.** Auto-rotates every 90 days with a 24h overlap
 window. Config override via `AUTH_JWT_KEY_ROTATION_DAYS` (S31).
- **Rate limits + SSRF + WAF baseline** defaulted on in every new project
 scaffold (S10).

### Docs

- **189 CLI wiki pages** audited against actual command behavior (v1.0.9
 snapshot). Zero drift entries at release time (S47, S48).
- **`web/org` + `web/docs` version strings auto-sourced** from
 `.claude/docs/MASTER-VERSIONS.md` at build time. No hand-written version
 numbers on marketing pages (S28, S29).
- **Changelog, pricing, plugin catalog** all driven by SPORT files via
 `web/org/scripts/sync-sport.mjs` (S42).

## [1.0.8] - 2026-04-16

P92 quick-fix batch. Interim release with Homebrew formula fixes, install script
refinements, and version banner updates.

### Fixed

- **`nself ai` default plugin URL.** `nself ai pool list`, `nself ai local
 status`, and the rest of the `ai` subcommands defaulted to
 `http://ai:3680`, which matched neither the real plugin-ai port (3709)
 nor the generated docker-compose service name (`plugin-ai`). Users had
 to set `PLUGIN_AI_INTERNAL_URL` manually to reach the plugin from
 inside compose. Default is now `http://plugin-ai:3709`; the env-var
 override still wins for non-standard deployments.

## [1.0.7] - 2026-04-16

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

## [1.0.6] - 2026-04-16

P89/P90 + P92 coordinated release. Ships P87-P90 CLI accumulated work alongside
plugins-pro v1.0.1.

### Added

- P89: goreleaser config for cross-platform binaries (proper release artifacts
 for macOS arm64/amd64, Linux amd64/arm64, Windows amd64).
- P89: monitoring docker-compose wiring for Prometheus/Grafana/Loki bundle
 properly wired into `nself build` output.
- P89: compliance checker command for auditing setup against security and
 configuration best practices.
- P89: expanded env var documentation across all generators.

### Fixed

- P89: homebrew formula release webhook. Correct sha256 computation and URL
 formatting on release; automated formula update now functions correctly.
- P90: assorted fixes and polish (see git log 5d232ec).

### Coordinated plugins-pro v1.0.1 P92

- Querier interface refactor. AI and Notify plugins now share a clean Querier
 abstraction with full handler test coverage.
- 8 Grafana dashboards. Drop-in dashboards for core services.
- Alertmanager rules. Pre-baked alert rules for common failure modes.
- BaseURL injection. Consistent config plumbing across services.
- 200-OK fixes and health tool wiring. All core endpoints now return expected codes.
- Hardening passes across AI, Google, Claw, Mux, Voice, Notify, and Browser plugins.

## [1.0.5] - 2026-04-15

### Highlights

- Docker DNS fix: plugin-to-plugin resolution now works via explicit compose network aliases
- Installer tarball path + filename alignment (fixes failed installs on some platforms)
- CLI error model: `os.Exit` calls replaced with returned errors across the command tree
- SSE buffer, signal handling, proxy CORS, and export pagination fixes
- 10 new `claw` subcommands for direct ɳClaw interaction
- Nginx resolver now uses the variable pattern everywhere, preventing Docker IP caching

### Added

- feat(claw): 10 CLI commands for interacting with ɳClaw (memory, topics, briefings,
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

## [1.0.4] - earlier

ɳClaw production-ready release with knowledge graph, agent dashboard, and 30+
community features.

### Highlights

- **Session identity fix.** Resolved root cause of production session issues
- **Knowledge graph.** Memory rooms, brain health scoring, Obsidian export
- **Prompt library.** 45+ templates with chaining support
- **Agent dashboard.** 8 personas with marketplace
- **Image generation.** Multi-provider support (DALL-E, Stable Diffusion, etc.)
- **Smart model routing.** User-defined rules for model selection
- **Chat UX overhaul.** Auto-titling, pinning, quick links, WebSocket streaming
- **PWA with offline support.** Full progressive web app capabilities
- **30 community-inspired features.** Voice input, keyboard shortcuts, conversation search, and more

## [1.0.3] - earlier

Security hardening release with expanded plugin ecosystem.

### Added

- 6 feature wiki pages (ɳClaw, ɳChat, ɳTV, ɳFamily, ɳCloud, Licensing Guide)
- API Reference covering all plugin REST endpoints
- Configuration with plugin environment variables (inter-plugin auth, AI routing, ɳClaw behavior, resource limits)
- Security Architecture with JWT model, passkey auth, ACL system, plugin-to-plugin auth, browser/shell/email security
- **`nself security` command.** New top-level command with `audit`, `setup`, and `status` subcommands for automated security hardening. Checks UFW, fail2ban, SSH, Docker port exposure, `.env` permissions, and service binding. See [[cmd-security]]
- **138 plugins.** The ecosystem now ships 25 free and 109 Pro plugins (up from 77 total in v1.0.0). New additions include **claw-budget** for AI token budget tracking and cost controls
- **New environment variables.** `CLAW_WEB_SECRET` for claw-web plugin authentication, `PLUGIN_INTERNAL_SECRET` for inter-plugin communication

### Security

- Container security improvements, stricter default `.env` permissions, and internal plugin authentication via `PLUGIN_INTERNAL_SECRET`

### Fixed

- Resolved edge cases in compose generation, improved error messages for license validation failures

### Upgrade

```bash
nself update
nself doctor    # verify stack health after upgrade
```

## [1.0.0] - 2026-03-29

First stable release. Complete Go rewrite of the ɳSelf CLI (previously Bash-based v0.x).

### Highlights

- 24 top-level commands with 295+ subcommands
- Full Docker Compose v2 stack generation (Postgres, Hasura, Auth, Nginx)
- Plugin system with 29 free plugins and 52 Pro plugins
- `nself migrate` command for v1 to v2 migration with rollback
- Interactive `nself init` wizard with multi-env support
- `nself health watch` for continuous monitoring
- Shell completion for bash, zsh, and fish
- `nself doctor` full system diagnostics
- License management for Pro plugins

---

Older entries are tracked in GitHub Releases.

[[Home]] | [[_Sidebar]]

## [1.0.10] - 2026-04-23 (Wave 5)

P94 Wave 5 patch. Unified auth complete (O01-O08), Claw BIOS Layers 1+2, plugin-sdk-go, audit log, rate-limit middleware, Let's Encrypt zero-config, pgvector default, self-healing schema, Ollama one-click, soak harness + badges + webhook outbox.

### Added

- **`nself account` top-level command** (O06), 7 subcommands: `login`, `logout`, `status`, `team`, `licenses`, `devices`, `transfer`. Manages ɳSelf account, sessions, licenses, team members, and devices.
- **`nself ollama` top-level command** (B38), 4 subcommands: `install`, `status`, `pull <model>`, `remove <model>`. One-click Ollama install wires `OLLAMA_BASE_URL` for the ai plugin automatically.
- **Zero-config Let's Encrypt** (B14), `nself build` auto-provisions Let's Encrypt cert on first deploy. Certbot renewal cron integrated.
- **pgvector default-on** (B23), `nself init` runs `CREATE EXTENSION IF NOT EXISTS vector`. `NSELF_PGVECTOR_ENABLED` defaults true.
- **Self-healing schema** (B37), `NSELF_SCHEMA_HEAL_ENABLED` flag. On `nself start`, diffs and auto-applies non-destructive migrations. Dry-run by default.
- **Webhook outbox** (R01-T07), `NSELF_WEBHOOK_OUTBOX_DIR` env var. Transactional webhook delivery via outbox table.
- **48h soak harness** (R01-T05), `cli/.github/workflows/soak.yml` canary traffic replay + latency regression gate.
- **CI status badges** (R01-T06), All repos updated via `.github/workflows/badges.yml`.



---

## [1.0.12] - 2026-04-25

P96 CRUNCH phase, golden-path E2E, release CLI, security hardening, plugin SDK migration, doctor/trust idempotency, and ship-readiness fixes.

### Added

- **`nself release` subcommand**: full 12-step release cascade automation, validates semver, coordinates CLI + admin + homebrew + ping_api + Docker Hub + registries in a single command (P96 T9).
- **Golden-path E2E test suite**: end-to-end scenario covering `nself init → build → start → plugin install → doctor → update → release` on a clean machine. Blocks CI on regression (P96 T10).
- **`nself doctor` coverage push**: 80%+ branch coverage on all doctor check functions; 100% on security-critical paths (P96 T10).
- **plugin-sdk-go migration**: all built-in plugin scaffolding and generated plugin stubs now reference the public `plugin-sdk-go` package (P96).
- **sport.json regen**: `nself release` triggers SPORT regeneration to keep F01-F15 ground-truth files in sync with the new binary (P96).

### Changed

- **Version bumped to v1.0.12** (lockstep with admin v1.0.12).
- **`cli/.github/VERSION`** updated from `1.0.10` to `1.0.12`.

### Fixed

- **`nself trust` / `nself dns-setup` / `nself ssl` idempotency**: state checks now bypass `osascript` entirely when target state is already configured. Eliminates stacked macOS admin dialogs in batch/CRUNCH contexts (P96 idempotency fix, see GCI Admin Prompt Hygiene).
- **License integrity validation**: `nself license verify` now checks Ed25519 signature against the bundled public key; tampered license files fail deterministically (P96 T4).
- **`nself install` UX**: progress output now streams line-by-line; spinner shows step name; error messages include remediation hint (P96 T9).

### Security

- **`nself doctor --security` hardening sweep** added to golden-path E2E. Runs SSRF guard, RLS audit, JWT key rotation check, and WAF config check, free tier, no license required.
