# nself bundle

> List, inspect, install, and remove ɳSelf plugin bundles.

## Synopsis

```
nself bundle <subcommand> [flags]
```

## Description

`nself bundle` manages plugin bundles. A bundle groups related plugins under a single license tier ($0.99/mo or $9.99/yr per bundle). Installing a bundle activates all plugins in it; removing it deactivates them.

Bundle install and remove are idempotent: running them twice produces no error and no duplicate state.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `list` | List all available bundles and their status |
| `info <bundle>` | Show bundle details: plugins, pricing, status |
| `install <bundle>` | Install a bundle (requires a valid license key) |
| `remove <bundle>` | Remove a bundle and deactivate its plugins |

## Available Bundles

| Bundle | System Name | Price |
|--------|-------------|-------|
| ɳClaw | `nclaw` | $0.99/mo or $9.99/yr |
| ɳChat | `nchat` | $0.99/mo or $9.99/yr |
| ɳFamily | `nfamily` | $0.99/mo or $9.99/yr |
| ɳTV | `ntv` | $0.99/mo or $9.99/yr |
| ClawDE | `clawde` | $0.99/mo or $9.99/yr |
| ɳSentry | `nsentry` | $0.99/mo or $9.99/yr |
| ɳSelf+ | `nself-plus` | $3.99/mo or $39.99/yr (all bundles) |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | — | Output in JSON format (for `list` and `info`) |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# List all bundles and their installed status
nself bundle list

# Show details for the ɳClaw bundle
nself bundle info nclaw

# Install the ɳClaw bundle (requires nself_claw_ license key)
nself bundle install nclaw

# Remove the ɳClaw bundle
nself bundle remove nclaw

# Install ɳSelf+ (unlocks all bundles)
nself bundle install nself-plus

# List bundles in JSON format (for scripting)
nself bundle list --json
```

## Idempotency

Both `install` and `remove` are safe to run multiple times:

- Installing an already-installed bundle: no-op, exits 0.
- Removing a bundle that is not installed: no-op, exits 0.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success (including no-op idempotent runs) |
| `1` | Error (license invalid, network failure, unknown bundle) |
| `2` | Misuse (missing required argument) |

## See Also

- [[cmd-license]], Manage plugin license keys
- [[cmd-costs]], Show estimated operational costs
- [[cmd-account]], Manage your ɳSelf account
