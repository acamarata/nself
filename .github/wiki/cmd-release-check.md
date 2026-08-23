# nself release-check

<!-- BEGIN PROSE:summary -->
> Run pre-flight validation before a release.
<!-- END PROSE:summary -->

## Synopsis

```
nself release-check <version> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself release-check` validates that the working environment is ready for a release
of the given version. It checks that the version string is well-formed, that the
target git tag does not already exist, that required credentials are present, and
that CI gates are passing.

`nself release` calls this command automatically as step 0 of the release cascade.
Run it independently to diagnose problems before committing to a full release run.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output results as JSON |
| `--skip-ci` | `false` | Skip GitHub CI check (use in offline environments) |
| `--skip-security` | `false` | Skip security CVE scan |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Validate that the environment is ready to release v1.2.0
nself release-check v1.2.0

# Run checks but skip CI status verification
nself release-check v1.2.0 --skip-ci

# Output results as JSON for scripting
nself release-check v1.2.0 --json
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-release]], run the full release cascade after checks pass
- [[cmd-release-status]], view live distribution surface status
- [[cmd-release-rollback]], roll back if a release needs reverting
- [[cmd-version]], check the currently installed CLI version
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
