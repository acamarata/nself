# Upgrading to v1.0.12

> Step-by-step guide for upgrading an nSelf installation from v1.0.11 to v1.0.12.

## Contents

- [Prerequisites](#prerequisites)
- [Upgrade steps](#upgrade-steps)
- [Breaking changes](#breaking-changes)
- [What changed in v1.0.12](#what-changed-in-v1012)
- [Known issues resolved](#known-issues-resolved)
- [Rollback](#rollback)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

Before upgrading, confirm you are on v1.0.11:

```bash
nself --version
# Expected: nself v1.0.11 (darwin/arm64) or linux/amd64
```

If you are on an older version, upgrade incrementally. Start from the guide for your current version:

- v0.9.x: [[Upgrade-From-v0.9]]
- v1.0.0 to v1.0.10: upgrade in-place using `nself upgrade`; no special migration required

Also confirm Docker is running:

```bash
docker info > /dev/null 2>&1 && echo "ok" || echo "Docker is not running"
```

---

## Upgrade steps

### 1. Back up your project

```bash
nself backup
```

This writes a timestamped backup to `~/.nself/backups/`. Keep it until you have verified the upgraded stack.

### 2. Install v1.0.12

**Homebrew (macOS/Linux):**

```bash
brew upgrade nself-org/nself/nself
nself --version
# Expected: nself v1.0.12
```

**Direct install (Linux):**

```bash
nself upgrade
nself --version
# Expected: nself v1.0.12
```

**Specific version pin (if needed):**

```bash
nself upgrade --version 1.0.12
```

### 3. Rebuild your project configuration

```bash
cd /path/to/your/project
nself build
```

This regenerates `docker-compose.yml` and nginx configs. The `--preview` flag on `nself plugin install` is new in this version if you want to inspect dependency trees before installing.

### 4. Restart the stack

```bash
nself stop
nself start
```

### 5. Run diagnostics

```bash
nself doctor
```

Address any warnings before moving on. The `nself doctor --security` sweep is now included in the default run path.

---

## Breaking changes

v1.0.12 has no breaking changes to CLI flags, environment variables, or plugin APIs. The upgrade is a drop-in replacement for v1.0.11.

One behavior change to be aware of:

- **`nself trust` / `nself dns-setup` / `nself ssl` are now idempotent.** If the target state is already configured, these commands exit immediately without prompting for macOS admin credentials. If you were scripting around the old behavior (e.g., always expecting a password prompt), your scripts will work correctly but will skip the dialog when the state is already set.

---

## What changed in v1.0.12

### Added

- **`nself release` command**: automates the 12-step release cascade (semver validation, CLI + admin + homebrew + ping_api + Docker Hub + registries coordination).
- **Golden-path E2E test suite**: end-to-end test covering `init → build → start → plugin install → doctor → update → release`. Blocks CI on regression.
- **`nself plugin install --preview`**: shows the dependency tree before installation. Useful for confirming what will be installed without committing.
- **`nself doctor` coverage**: 80%+ branch coverage on all check functions; 100% on security-critical paths.
- **Plugin SDK migration**: plugin stubs and scaffolding now reference the public `plugin-sdk-go` package.

### Fixed

- **`nself trust` / `nself dns-setup` / `nself ssl` idempotency**: state checks now bypass the macOS `osascript` admin dialog when the target state is already configured. Eliminates stacked password prompts in automation and CI contexts.
- **License integrity**: `nself license verify` now checks Ed25519 signature against the bundled public key. Tampered license files fail deterministically.
- **`nself install` progress output**: streams line-by-line, spinner shows current step name, error messages include remediation hints.
- **`nself doctor` SSRF/RLS/JWT/WAF checks**: now run as part of `--security` sweep with no license requirement.
- **Test suite stability**: resolved scaffold, clone, and harness test failures. `TestErrorHarness_AllCommandsCovered` is now green.
- **Monitoring compose blocks**: added CPU limits to Tempo and OTEL collector blocks to prevent resource saturation.

### Security

- **`nself doctor --security` hardening sweep** added to golden-path E2E. Covers SSRF guard, RLS audit, JWT key rotation check, and WAF config check. Free tier, no license required.

---

## Known issues resolved

| Issue | Fixed in |
|---|---|
| macOS admin dialog appeared repeatedly in CI and batch contexts when running `nself trust` or `nself dns-setup` | v1.0.12 |
| `nself license verify` did not check file integrity; tampered license files were accepted | v1.0.12 |
| `nself install` output was buffered; on slow connections there was no visible progress | v1.0.12 |
| `go test ./...` failures in scaffold and clone tests (`TestScaffold_*`, `TestClone_*`) caused CI failures on `main` | v1.0.12 |

---

## Rollback

If you need to revert to v1.0.11 after upgrading:

**Using the built-in rollback (direct install only):**

```bash
nself upgrade --rollback
nself --version
# Expected: nself v1.0.11
```

**Using a specific version pin:**

```bash
nself upgrade --version 1.0.11
```

**Using Homebrew:**

```bash
brew install nself-org/nself/nself@1.0.11
```

After rollback, restore your project backup:

```bash
nself restore ~/.nself/backups/<timestamp>.tar.gz
```

---

## Troubleshooting

### `nself --version` shows 1.0.11 after upgrade

The binary was not replaced. Check your `$PATH`:

```bash
which nself
# Should point to /opt/homebrew/bin/nself or /usr/local/bin/nself
```

If you have multiple nself binaries, remove stale copies.

### `nself build` fails after upgrade

Run with verbose output:

```bash
nself build --verbose
```

Common causes: Docker not running, stale `.env.computed` file, port conflict on 5432 or 8080. Check `nself doctor` first.

### `nself doctor` reports license integrity failure

Your license file may have been corrupted. Re-enter your key:

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself license verify
```

---

See also:

- [[cmd-upgrade]] — full `nself upgrade` command reference
- [[Changelog]] — full release history
- [[Installation]] — fresh install guide
- [[Upgrade-From-v0.9]] — migrating from the legacy Bash CLI
- [[Upgrading]] — upgrade guide index

← [[Installation]] | [[Home]] →
