# nself release-rollback

<!-- BEGIN PROSE:summary -->
> Roll back distribution surfaces to a prior release version.
<!-- END PROSE:summary -->

## Synopsis

```
nself release-rollback <version> <prior-version> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself release-rollback` reverts the Homebrew tap formula, ping_api environment,
and Docker Admin image tag to a previously published version. It also writes a
changelog entry recording the rollback.

This command does not roll back production deployments or Vercel subapps. Use
`nself deploy rollback` for live deployment rollback.

Deleting git tags is opt-in and destructive. Pass `--delete-tags` only when you
need to retract a version from GitHub Releases entirely.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--delete-tags` | `false` | Also delete git tags (DESTRUCTIVE — requires explicit flag) |
| `--dry-run` | `false` | Log rollback steps without executing |
| `--json` | `false` | Output as JSON |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Roll back v1.2.0 and restore v1.1.9 across distribution surfaces
nself release-rollback v1.2.0 v1.1.9

# Rehearse a rollback without touching anything
nself release-rollback v1.2.0 v1.1.9 --dry-run
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-release]], run the full release cascade
- [[cmd-release-check]], pre-flight validation before releasing
- [[cmd-release-status]], check which surfaces are live
- [[cmd-deploy]], roll back live production deployments
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
