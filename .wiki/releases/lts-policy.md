# nself LTS Policy

## What is LTS?

Long-Term Support (LTS) releases receive security patches, critical bug fixes, and compatibility updates for a fixed period. This lets teams deploy nself with confidence that their infrastructure will remain stable and supported.

## v1.0 LTS

- **Support period:** 2 years from release date
- **End of life:** TBD (release date + 2 years)
- **Guaranteed updates:** security patches, critical bugs, Bash/Docker compat
- **No breaking changes** in the v1.x series

## What LTS Covers

| Category | Covered | Example |
|----------|---------|---------|
| Security vulnerabilities | Yes | CVE in a dependency, auth bypass |
| Critical bugs | Yes | Data loss, service crash, build failure |
| Bash compatibility | Yes | macOS ships new Bash version |
| Docker compatibility | Yes | Docker Engine deprecates a feature we use |
| New features | No | Added in v1.x minor releases, never breaking |
| Performance improvements | Best effort | Backported if low-risk |
| Plugin API changes | No breaking changes | New plugin hooks added, existing ones stable |

## Version Numbering

nself follows semantic versioning within LTS:

- **v1.0.x** - patch releases (bug fixes, security)
- **v1.x.0** - minor releases (new features, non-breaking)
- **v2.0.0** - next major (may have breaking changes, separate LTS cycle)

## Support Channels

- **GitHub Issues:** github.com/nself-org/cli/issues (all tiers)
- **Community:** nself.org/community (all tiers)
- **Email support:** Elite tier and above
- **24h response:** Business tier and above
- **Dedicated channel:** Business+ and Enterprise

## Upgrade Path

When v1.0 LTS reaches end of life, users can:

1. **Upgrade to the next LTS** (recommended) - migration guide provided
2. **Stay on v1.0** - continues to work, just no more patches
3. **Pin a specific version** - `brew install nself@1.0` or pin in install.sh

There will always be at least 6 months of overlap between an LTS end-of-life and the next LTS release, so teams have time to plan upgrades.
