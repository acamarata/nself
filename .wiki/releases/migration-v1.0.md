# Migrating from v0.9.x to v1.0

This guide covers upgrading from nself v0.9.9 (any RC) to v1.0 LTS.

## Quick Upgrade

```bash
brew upgrade nself
# or
curl -fsSL https://install.nself.org | bash
```

Then rebuild your stack:

```bash
cd /path/to/your/project
nself build
nself restart
```

## Breaking Changes

**None.** v1.0 is a stability release. All v0.9.9 configurations, plugins, and `.env` files work without modification.

## What Changed

### Plugin Licensing

The licensing system moved from 2-tier (Pro/Max) to 6-tier membership:

| Old Tier | New Tier | Price |
|----------|----------|-------|
| Free | Free | $0 |
| Pro ($9.99/yr) | Basic ($0.99/mo or $9.99/yr) | Same annual price |
| Max ($19.99/yr) | Pro ($1.99/mo or $19.99/yr) | Same annual price |
| (new) | Elite | $4.99/mo or $49.99/yr |
| (new) | Business | $9.99/mo or $99.99/yr |
| (new) | Business+ | $49.99/mo or $499.99/yr |
| (new) | Enterprise | $99.99/mo or $999.99/yr |

Existing license keys are automatically grandfathered. No action needed.

### New Commands

No new top-level commands. New subcommands added:

- `nself license tier` - show your current membership tier
- `nself plugin search` - search the plugin marketplace
- `nself doctor --deep` - extended health check

### Removed Commands

None. All v0.9.x commands work exactly the same.

### Environment Variables

No env var changes. All existing `.env` files work without modification.

### Docker Compose

`nself build` generates the same compose structure. No manual docker-compose changes needed.

### Plugin System

- Plugin manifests now include a `tier` field (Basic, Pro, etc.)
- `nself plugin install` checks tier entitlement server-side
- Existing installed plugins continue to work

## Verification

After upgrading, verify your installation:

```bash
nself version          # should show v1.0.0
nself doctor           # should report all-green
nself status           # should show all services running
```

## LTS Commitment

v1.0 is a Long-Term Support release with a 2-year commitment:

- Security patches for 2 years from release date
- Critical bug fixes for 2 years
- Bash and Docker compatibility maintained
- No breaking changes in the v1.x series

See docs.nself.org/lts for the full LTS policy.

## Getting Help

- Documentation: docs.nself.org
- Issues: github.com/nself-org/cli/issues
- Community: nself.org/community
