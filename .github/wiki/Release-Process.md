# Release Process

How ɳSelf CLI releases are built, tagged, and distributed.

## Version Files

Two files must be updated for a release:

| File | Content |
|------|---------|
| `.github/VERSION` | Plain text version string, e.g. `1.0.0` |
| `internal/version/version.go` | Go constant: `const Version = "1.0.0"` |

Both must match exactly before tagging.

## Tag Format

Releases use semantic versioning with a `v` prefix:

```
v1.0.0       # stable release
v1.0.1       # patch release
v1.1.0       # minor release
v2.0.0       # major release
v1.0.0-rc.1  # release candidate
```

## Release Workflow

The CI release workflow (`.github/workflows/release.yml`) triggers automatically on a version tag push:

```bash
# 1. Update version files
# 2. Commit
git commit -m "release: v1.0.0"

# 3. Tag
git tag v1.0.0

# 4. Push tag (triggers CI)
git push origin v1.0.0
```

The workflow:
1. Runs tests and linting
2. Cross-compiles for all targets (Linux/macOS/Windows x amd64/arm64)
3. Creates release archives (`.tar.gz` and `.zip`)
4. Uploads artifacts to the GitHub Release
5. Generates release notes from commit history

## Homebrew Formula

The `homebrew-nself` tap auto-updates when a new GitHub Release is published. No manual action required, the formula reads the latest release tag from GitHub.

**Homebrew lockstep gate:** The workflow `.github/workflows/homebrew-version-lockstep.yml` runs on every `v*.*.*` tag push. It reads the Homebrew formula version from `nself-org/homebrew-nself/Formula/nself.rb` via the GitHub API and fails if the formula version does not match the tag being cut. If the gate fails: update the formula in `homebrew-nself` first (version string + sha256), then re-tag on `cli`. Never cut a CLI tag while the formula is behind. This gate closes the gap that allowed v1.0.6/v1.0.7/v1.0.8 to ship without Homebrew formula sync (S30-T02).

## install.sh

The installer at `install.nself.org` is hosted in a separate repo. Update the version reference in `install.sh` after the GitHub Release is live.

## Release Announcement

After the release:
1. Write GitHub Release notes highlighting new features and breaking changes
2. Update [[Changelog]] with the release summary
3. Post to ɳSelf community channels

## See Also

- [[Contributing]], how to contribute code
- [[Dev-Setup]], local development setup
- [[Changelog]], version history

---
← [[Home]] | [[_Sidebar]]
