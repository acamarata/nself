# Contributing to nself

Thank you for helping improve nself. This guide covers everything you need to contribute code, tests, or documentation.

## Table of contents

- [Development setup](#development-setup)
- [Making changes](#making-changes)
- [Tests](#tests)
- [Bash 3.2 compatibility rules](#bash-32-compatibility-rules)
- [ShellCheck requirements](#shellcheck-requirements)
- [PR process](#pr-process)
- [Release process](#release-process)

---

## Development setup

**Prerequisites:**

- Bash 3.2 or newer (macOS ships with 3.2; Linux typically has 5.x)
- Docker 24+ with Docker Compose v2
- [bats-core](https://github.com/bats-core/bats-core) 1.8+ for running tests
- [ShellCheck](https://www.shellcheck.net/) for linting

**Install bats:**

```bash
# macOS
brew install bats-core

# Ubuntu/Debian
sudo apt-get install bats

# Or directly
git clone https://github.com/bats-core/bats-core.git && cd bats-core && ./install.sh /usr/local
```

**Clone and verify:**

```bash
git clone https://github.com/nself-org/cli.git
cd cli

# Verify CLI works
bash bin/nself --version

# Run the full test suite
bats src/tests/
```

---

## Making changes

All functionality lives in `src/cli/` (top-level commands) and `src/lib/` (shared libraries).

**Structure:**

```
src/
  cli/         # One file per top-level command (init.sh, start.sh, plugin.sh, ...)
  lib/         # Shared libraries grouped by domain
    utils/     # Cross-platform wrappers (platform-compat.sh)
    build/     # docker-compose generation
    plugin/    # Plugin system runtime
    ...
```

**Adding a subcommand:**

New functionality must be a subcommand of an existing top-level command (`db`, `plugin`, `service`, `deploy`, etc.). New top-level commands require explicit maintainer approval and must cover a completely new domain with 5+ subcommands.

**Code-docs-commit workflow:**

1. Implement the feature in `src/cli/` or `src/lib/`
2. Update `.wiki/commands/<category>/README.md`
3. Update help text in the CLI file (the `--help` output)
4. Update `COMMAND-TREE-V1.md` if the command structure changed
5. Test: `bash src/cli/<command>.sh --help` and run integration tests
6. Commit code and docs together in one commit

---

## Tests

Run the full test suite:

```bash
bats src/tests/
```

Run a specific test file:

```bash
bats src/tests/unit/test-init.sh
bats src/tests/integration/test-plugin.sh
```

**Requirements:**

- Every new command or flag needs a corresponding bats test
- Integration tests must pass with both a fresh install and an existing project
- Tests must not require network access (mock external calls)

---

## Bash 3.2 compatibility rules

This is the most important rule. CI runs on Bash 3.2 (macOS default). Violations break CI.

| Forbidden | Use instead |
|-----------|-------------|
| `echo -e "..."` | `printf "...\n"` |
| `${var,,}` (lowercase) | `printf '%s' "$var" \| tr '[:upper:]' '[:lower:]'` |
| `${var^^}` (uppercase) | `printf '%s' "$var" \| tr '[:lower:]' '[:upper:]'` |
| `declare -A` (associative arrays) | Parallel arrays or `case` statements |
| `mapfile` / `readarray` | `while IFS= read -r line` loops |
| `stat -c` / `stat -f` (raw) | `safe_stat_perms()` from `platform-compat.sh` |
| `sed -i ''` vs `sed -i` | `safe_sed_inline()` from `platform-compat.sh` |
| `readlink -f` | `safe_readlink()` from `platform-compat.sh` |

**Verify before committing:**

```bash
# No echo -e
grep -rn 'echo -e' src/

# No Bash 4+ features
grep -r '\${[^}]*,,}' src/
grep -r '\${[^}]*\^\^}' src/
grep -r 'declare -A' src/
grep -rE '\b(mapfile|readarray)\b' src/
```

---

## ShellCheck requirements

All shell files must pass ShellCheck at error severity:

```bash
shellcheck -S error src/cli/*.sh
shellcheck -S error src/lib/**/*.sh
```

ShellCheck runs in CI. PRs with warnings will not be merged.

---

## PR process

1. Fork the repo and create a branch: `git checkout -b feat/my-feature`
2. Make your changes following the rules above
3. Run tests: `bats src/tests/`
4. Run ShellCheck: `shellcheck -S error src/cli/*.sh`
5. Open a PR using the pull request template

**PR checklist** (also in the PR template):

- [ ] Tests added for new behavior
- [ ] `bats src/tests/` passes locally
- [ ] ShellCheck clean (`shellcheck -S error src/cli/*.sh`)
- [ ] Docs updated (`.wiki/commands/` and help text)
- [ ] No Bash 4+ syntax (verify with the grep commands above)

---

## Release process

Releases are made by maintainers. The process:

1. Update version in all version files (see `cli/.claude/CLAUDE.md` for the full list)
2. Update `.wiki/releases/vX.Y.Z.md`, `CHANGELOG.md`, `README.md`
3. Commit: `git commit -m "release: vX.Y.Z - Title"`
4. Tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z - Title"`
5. Push code and tag: `git push && git push --tags`
6. Create GitHub Release — Homebrew tap auto-syncs from the release tarball

---

## Questions?

Open a [GitHub Discussion](https://github.com/nself-org/cli/discussions) or join [Discord](https://discord.gg/nself).
