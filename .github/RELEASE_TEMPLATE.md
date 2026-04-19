# Release vX.Y.Z

**Channel:** stable | rc | edge
**Date:** YYYY-MM-DD
**Matching artifacts:** admin vX.Y.Z · homebrew formula vX.Y.Z

## Summary

One-paragraph description of what this release contains. Focus on user-visible change. Write for a skeptical developer who wants to know "what's in it and why should I upgrade."

## Highlights

- Short bullet — user-visible feature, fix, or improvement
- Another bullet
- Another bullet

## New Commands / Flags

| Command | What it does |
|---|---|
| `nself <cmd>` | One-liner |

Leave this section out if no new commands.

## Breaking Changes

If none: write "None." If any: list each one with a before / after example and migration steps. A patch release should have NONE. A minor release MUST call them out prominently.

## Security

Any CVE fixes or hardening changes. Reference advisory IDs if published. nSelf security is always free — no paywalled fixes.

## Deprecations

Anything newly marked deprecated. Include removal timeline (typically two minor versions out).

## Bug Fixes

- Fixed X where Y happened
- Fixed Z in corner case W

## Internal / Non-user-visible

Only include if the change is large enough that downstream maintainers need to know (e.g., loader order changes, schema changes, build toolchain version bump).

## Install

```bash
# Homebrew
brew upgrade nself

# Direct binary
curl -sL https://github.com/nself-org/cli/releases/download/vX.Y.Z/nself-X.Y.Z-$(uname -s | tr A-Z a-z)-$(uname -m).tar.gz | tar -xz
sudo mv nself /usr/local/bin/

# Self-upgrade from existing install
nself upgrade
```

## Verify

Every release artifact is signed with Sigstore keyless signing. To verify:

```bash
cosign verify-blob \
  --bundle <tarball>.tar.gz.sig \
  --certificate-identity-regexp '^https://github.com/nself-org/cli/\.github/workflows/release\.yml@refs/tags/vX.Y.Z$' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  <tarball>.tar.gz
```

Full signing + SBOM + provenance details: [release-signing.md](https://github.com/nself-org/nself/blob/main/.claude/docs/operations/release-signing.md).

## Artifacts

Every release includes:

- 4 platform tarballs (`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`)
- `checksums.txt` — SHA-256 of all tarballs
- `sbom.spdx.json` + per-tarball SBOMs (SPDX format)
- `provenance.intoto.jsonl` — SLSA v1.0 build provenance
- `*.sig` — cosign signature bundles for every artifact above

## Upgrade Notes

Anything users need to do when upgrading (migration commands, config changes, plugin re-install). If nothing: write "Drop-in upgrade — no action required."

## Known Issues

- Short list of issues we know about but did not block this release on. Reference GitHub issue numbers.

---

**Template usage:** Copy this file into `release-body.md` at release time, fill in every section, delete unused sections, paste into the GitHub Release body (or let the release workflow auto-generate from `.github/wiki/Changelog.md`. see `cli/.github/workflows/release.yml` S34-T09).
