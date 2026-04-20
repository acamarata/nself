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

**v1.0.9 ships no telemetry client.** No data is collected or transmitted by the CLI at this version. This command exists so you can set your preference now, before the v1.1.0 client ships.

When the v1.1.0 telemetry client ships, it will collect only anonymous aggregate data (CLI version, platform, architecture) for usage analysis. All collection is opt-in, and your stored preference is respected.

See [Privacy Policy](https://nself.org/legal/privacy) for full details.

## Preference Resolution (Priority Order)

1. `NSELF_TELEMETRY_OPT_OUT=1` environment variable (highest priority)
2. `~/.nself/config.toml` `[telemetry]` section
3. Default: enabled (no effect at v1.0.9)

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
Telemetry: enabled
Source:    default

v1.0.9: no telemetry client ships. This preference will take effect in v1.1.0.
Privacy policy: https://nself.org/legal/privacy
```

**After running `nself telemetry off`:**

```
Telemetry: disabled (opt-out)
Source:    config

v1.0.9: no telemetry client ships. This preference will take effect in v1.1.0.
Privacy policy: https://nself.org/legal/privacy
```

## Config File Format

Preferences are stored in `~/.nself/config.toml`:

```toml
[telemetry]
enabled = false
```

This file is shared with other nSelf user-level settings. It is created automatically when you run `nself telemetry off` or `nself telemetry on`.

## Privacy

All security features in nSelf are free. When telemetry ships in v1.1.0, it will be:

- Opt-in only
- Anonymous (no personal data, no project names, no IP addresses stored)
- Transparent (exact fields documented in the Privacy Policy before collection begins)

← [[Commands]] | [[Home]] →
