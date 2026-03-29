# Release Process

How nSelf CLI releases are built, tagged, and distributed.

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

The `homebrew-nself` tap auto-updates when a new GitHub Release is published. No manual action required — the formula reads the latest release tag from GitHub.

## install.sh

The installer at `install.nself.org` is hosted in a separate repo. Update the version reference in `install.sh` after the GitHub Release is live.

## Release Announcement

After the release:
1. Write GitHub Release notes highlighting new features and breaking changes
2. Update [[Changelog]] with the release summary
3. Post to nSelf community channels

## See Also

- [[Contributing]] — how to contribute code
- [[Dev-Setup]] — local development setup
- [[Changelog]] — version history

---
← [[Home]] | [[_Sidebar]]
