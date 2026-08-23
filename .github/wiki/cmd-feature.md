# nself feature

<!-- BEGIN PROSE:summary -->
> Manage CLI-built-in feature flags for build-time and install-time capability gates.
<!-- END PROSE:summary -->

## Synopsis

```
nself feature <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself feature` exposes the binary-local feature flag registry compiled into the nSelf CLI. It is distinct from `nself flag`, which manages runtime feature flags via the feature-flags plugin on port 3305. The `feature` subcommand covers build-time and install-time capability gates for v1.1.0+ release features such as ɳSentry, ɳFamily COPPA strict mode, and multi-tenant strict mode.

The registry is defined in `internal/featureflags/registry.yaml` and compiled into the binary. Per-project overrides persist to `.nself/features.json` in the current working directory, which nSelf treats as the project root (matching `nself build` and `nself start` behavior).

`list` prints every registered flag with its effective enabled state (override or registry default), the source of the effective value, the flag type, the surface it applies to, and the version that introduced it. Pass `--json` for machine-readable output.

`enable <flag>` and `disable <flag>` record an override in `.nself/features.json`. They require exactly one positional argument: the flag key. The change is immediate and persists across CLI invocations for that project directory.

`status <flag>` shows the full registry entry and effective state for one flag. Pass `--json` to get the structured representation. It exits with an error if the flag key is not registered.

### `feature list`
### `feature enable`
No flags. Requires exactly one positional argument: the flag key.
### `feature disable`
No flags. Requires exactly one positional argument: the flag key.
### `feature status`
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `disable` | Disable a feature flag (records an override) |
| `enable` | Enable a feature flag (records an override) |
| `list` | List all registered feature flags with effective state |
| `status` | Show the effective state for a single flag |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# List all registered flags in table format
nself feature list
```

```bash
# List all flags as JSON
nself feature list --json
```

```bash
# Enable a specific flag for this project
nself feature enable nsentry-enabled
```

```bash
# Disable a flag
nself feature disable clawde-telemetry-opt-in
```

```bash
# Check the effective state of one flag
nself feature status nfamily-coppa-strict
```

```bash
# Check a flag's state as JSON
nself feature status nsentry-enabled --json
```

```bash
# Enable ɳFamily COPPA strict mode
nself feature enable nfamily-coppa-strict
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [cmd-flag.md](cmd-flag.md) — runtime feature-flag management via the feature-flags plugin (port 3305)
- [cmd-build.md](cmd-build.md) — build pipeline that reads feature gates at build time
- [cmd-plugin.md](cmd-plugin.md) — install plugins that gate features behind a license
- [cmd-doctor.md](cmd-doctor.md) — verify configuration and detect misconfigured flags
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
