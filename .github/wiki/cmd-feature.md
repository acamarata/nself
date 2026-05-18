# nself feature

> Manage CLI-built-in feature flags for build-time and install-time capability gates.

## Synopsis

```
nself feature <subcommand> [flags]
nself feature list [--json]
nself feature enable <flag>
nself feature disable <flag>
nself feature status <flag> [--json]
```

## Description

`nself feature` exposes the binary-local feature flag registry compiled into the nSelf CLI. It is distinct from `nself flag`, which manages runtime feature flags via the feature-flags plugin on port 3305. The `feature` subcommand covers build-time and install-time capability gates for v1.1.0+ release features such as ɳSentry, ɳFamily COPPA strict mode, and multi-tenant strict mode.

The registry is defined in `internal/featureflags/registry.yaml` and compiled into the binary. Per-project overrides persist to `.nself/features.json` in the current working directory, which nSelf treats as the project root (matching `nself build` and `nself start` behavior).

`list` prints every registered flag with its effective enabled state (override or registry default), the source of the effective value, the flag type, the surface it applies to, and the version that introduced it. Pass `--json` for machine-readable output.

`enable <flag>` and `disable <flag>` record an override in `.nself/features.json`. They require exactly one positional argument: the flag key. The change is immediate and persists across CLI invocations for that project directory.

`status <flag>` shows the full registry entry and effective state for one flag. Pass `--json` to get the structured representation. It exits with an error if the flag key is not registered.

## Flags

### `feature list`

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--json` | `–` | bool | false | Output as JSON |

### `feature enable`

No flags. Requires exactly one positional argument: the flag key.

### `feature disable`

No flags. Requires exactly one positional argument: the flag key.

### `feature status`

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--json` | `–` | bool | false | Output as JSON |

## Examples

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

## See Also

- [cmd-flag.md](cmd-flag.md) — runtime feature-flag management via the feature-flags plugin (port 3305)
- [cmd-build.md](cmd-build.md) — build pipeline that reads feature gates at build time
- [cmd-plugin.md](cmd-plugin.md) — install plugins that gate features behind a license
- [cmd-doctor.md](cmd-doctor.md) — verify configuration and detect misconfigured flags

← [[Commands]] | [[Home]] →
