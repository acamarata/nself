# Plugin Status Badges

Every plugin in the nSelf registry carries a status field that reflects its production readiness. The CLI displays status badges in `nself plugin list` output and enforces install-time behavior based on that status.

## Status Values

| Status | Badge in `list` | Meaning |
|--------|-----------------|---------|
| `stable` | _(no badge)_ | Production-ready, tested, supported |
| `beta` | `[beta]` | Functional but may have rough edges; not yet production-supported |
| `planned` | `[planned]` | Listed in the registry, not yet available to install |

## Behavior at Install Time

### stable

Installs without any warning. This is the expected state for any plugin you run in production.

```
$ nself plugin install ai
Downloading ai v1.2.0...
Verified sha256 checksum.
Plugin ai installed.
```

### beta

Installs with a warning printed to stderr. The install proceeds normally — you can use the plugin, but exercise caution in production environments.

```
$ nself plugin install records
[beta] records is in beta — use in production at your own risk
Downloading records v0.3.0...
Verified sha256 checksum.
Plugin records installed.
```

### planned

Install is rejected. The plugin is on the roadmap but the tarball is not yet published.

```
$ nself plugin install nfamily
Error: plugin "nfamily" is not yet available — coming soon.
See https://nself.org/plugins/nfamily for the release timeline.
Run 'nself plugin list' to see available plugins.
```

## Viewing Status in `nself plugin list`

Stable plugins show no badge. Beta and planned plugins show their badge inline:

```
ai                  [installed]
analytics           [installed]
browser             [beta]
claw                [installed]
nfamily             [planned]
```

## Upgrading from Beta to Stable

When a plugin graduates from beta to stable, the registry update is automatic. Run `nself plugin refresh` to pull the latest registry before upgrading:

```bash
nself plugin refresh
nself plugin update records
```

## For Plugin Developers

Plugin status is set in `plugin.json`:

```json
{
  "name": "records",
  "status": "beta"
}
```

Valid values: `stable`, `beta`, `planned`. Omitting the field defaults to `stable`.

`planned` plugins should have a `plugin.json` and a registry entry but no published tarball. This allows the CLI to give useful guidance instead of a generic "plugin not found" error.

## Related Pages

- [[Plugin-Overview]] — Plugin tiers and pricing
- [[Plugin-Install]] — Full install workflow
- [[cmd-plugin]] — `nself plugin` command reference

← [[Plugin-Overview]] | [[Home]] →
