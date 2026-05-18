# nself release-check

> Run pre-flight validation before a release.

## Synopsis

```bash
nself release-check <version> [flags]
```

## Description

`nself release-check` validates that the working environment is ready for a release
of the given version. It checks that the version string is well-formed, that the
target git tag does not already exist, that required credentials are present, and
that CI gates are passing.

`nself release` calls this command automatically as step 0 of the release cascade.
Run it independently to diagnose problems before committing to a full release run.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--json` | `–` | bool | `false` | Output results as JSON |
| `--skip-ci` | `–` | bool | `false` | Skip GitHub CI check (use in offline environments) |
| `--skip-security` | `–` | bool | `false` | Skip security CVE scan |

## Examples

```bash
# Validate that the environment is ready to release v1.2.0
nself release-check v1.2.0

# Run checks but skip CI status verification
nself release-check v1.2.0 --skip-ci

# Output results as JSON for scripting
nself release-check v1.2.0 --json
```

## See Also

- [[cmd-release]], run the full release cascade after checks pass
- [[cmd-release-status]], view live distribution surface status
- [[cmd-release-rollback]], roll back if a release needs reverting
- [[cmd-version]], check the currently installed CLI version

← [[Commands]] | [[Home]] →
