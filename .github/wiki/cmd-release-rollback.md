# nself release-rollback

> Roll back distribution surfaces to a prior release version.

## Synopsis

```bash
nself release-rollback <version> <prior-version> [flags]
```

## Description

`nself release-rollback` reverts the Homebrew tap formula, ping_api environment,
and Docker Admin image tag to a previously published version. It also writes a
changelog entry recording the rollback.

This command does not roll back production deployments or Vercel subapps. Use
`nself deploy rollback` for live deployment rollback.

Deleting git tags is opt-in and destructive. Pass `--delete-tags` only when you
need to retract a version from GitHub Releases entirely.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dry-run` | `–` | bool | `false` | Simulate all rollback steps without making external changes |
| `--json` | `–` | bool | `false` | Emit structured JSON output instead of human-readable progress |
| `--delete-tags` | `–` | bool | `false` | Delete git tags for `<version>` after rolling back (destructive) |

## Examples

```bash
# Roll back v1.2.0 and restore v1.1.9 across distribution surfaces
nself release-rollback v1.2.0 v1.1.9

# Rehearse a rollback without touching anything
nself release-rollback v1.2.0 v1.1.9 --dry-run
```

## See Also

- [[cmd-release]], run the full release cascade
- [[cmd-release-check]], pre-flight validation before releasing
- [[cmd-release-status]], check which surfaces are live
- [[cmd-deploy]], roll back live production deployments

← [[Commands]] | [[Home]] →
