# Upgrading

Upgrade guides for moving between nSelf CLI versions.

## Current release

| From | To | Guide |
|---|---|---|
| v1.0.11 | **v1.0.12** | [[Upgrading-v1.0.12]] |

## Previous releases

| From | To | Guide |
|---|---|---|
| v1.0.x (Bash era) | v1.0.0+ (Go CLI) | [[Upgrading-from-v1]] |
| v0.9.x | v1.0.9 | [[Upgrade-From-v0.9]] |

## General upgrade procedure (patch versions)

For any patch upgrade within the same minor (e.g. v1.0.11 to v1.0.12):

```bash
# macOS (Homebrew)
brew upgrade nself-org/nself/nself

# Linux (direct install)
nself upgrade

# Verify
nself --version
```

After upgrading, rebuild and restart your project:

```bash
nself build
nself stop && nself start
nself doctor
```

## Rollback

The `nself upgrade --rollback` flag reverts the binary to the previous installed version (direct-install only). Homebrew users pin with `brew install nself-org/nself/nself@<version>`.

---

[[Home]] | [[Installation]] | [[Changelog]]
