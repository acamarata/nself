# nself telemetry

> Manage CLI telemetry opt-in/out preferences.

## Synopsis

```
nself telemetry <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
|-----------|-------------|
| `status` | Show the current telemetry opt-out state |
| `off` | Opt out of telemetry (writes to `~/.nself/config.toml`) |
| `on` | Opt in to telemetry (writes to `~/.nself/config.toml`) |

## Description

`nself telemetry` manages your preference for CLI usage telemetry.

**v1.1.1 ships the opt-in telemetry client.** Collection is OFF by default; you must explicitly opt in. The client collects only anonymous aggregate data (CLI version, platform, architecture, command counts) for usage analysis. All collection respects your stored preference.

See [Privacy Policy](https://nself.org/legal/privacy) for full details.

## Preference Resolution (Priority Order)

1. `NSELF_TELEMETRY_OPT_OUT=1` environment variable (highest priority)
2. `~/.nself/config.toml` `[telemetry]` section
3. Default: disabled (opt-in)

The environment variable always wins. Use it for CI/CD pipelines or shared servers where you do not want to modify the config file.

## Examples

```bash
# Check current status
nself telemetry status

# Opt out permanently (writes to ~/.nself/config.toml)
nself telemetry off

# Opt back in
nself telemetry on

# Opt out via environment variable (no file change)
NSELF_TELEMETRY_OPT_OUT=1 nself telemetry status
```

**Sample output (status):**

```
Telemetry: disabled (default)
Source:    default

v1.1.1: opt-in telemetry client. Run 'nself telemetry on' to enable.
Privacy policy: https://nself.org/legal/privacy
```

**After running `nself telemetry on`:**

```
Telemetry: enabled (opt-in)
Source:    config

v1.1.1: opt-in telemetry client active. Anonymous aggregate data only.
Privacy policy: https://nself.org/legal/privacy
```

## Config File Format

Preferences are stored in `~/.nself/config.toml`:

```toml
[telemetry]
enabled = false
```

This file is shared with other ɳSelf user-level settings. It is created automatically when you run `nself telemetry off` or `nself telemetry on`.

## Privacy

All security features in ɳSelf are free. When telemetry ships in v1.1.0, it will be:

- Opt-in only
- Anonymous (no personal data, no project names, no IP addresses stored)
- Transparent (exact fields documented in the Privacy Policy before collection begins)

← [[Commands]] | [[Home]] →
