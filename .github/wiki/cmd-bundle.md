# nself bundle

> List, inspect, install, and remove ɳSelf plugin bundles.

## Synopsis

```bash
nself bundle <subcommand> [flags]
nself bundle list
nself bundle info <bundle>
nself bundle install <bundle>
nself bundle remove <bundle>
```

## Description

`nself bundle` manages plugin bundles. A bundle is a named collection of paid plugins sold together under a single subscription. Installing a bundle activates its plugins for the life of the subscription.

The six current bundles are:

| Bundle | Slug | Plugins |
|--------|------|---------|
| ɳClaw | `nclaw` | AI assistant + memory + BYOK + claw-web + claw-news + claw-budget |
| ɳChat | `nchat` | Chat + e2ee + moderation + invitations + realtime + push |
| ɳFamily | `nfamily` | Family social + gedcom + calendar + tasks + photos + home |
| ɳTV | `ntv` | IPTV + EPG + streaming + retro-gaming + rom-discovery + media-processing |
| ClawDE | `clawde` | Dev environment + github-runner + mlflow + feature-flags + webhooks + compliance |
| ɳSentry | `nsentry` | Monitoring + alerting + anomaly + SLO + synthetic + RUM + crash + oncall |

ɳSelf+ (`nself-plus`) grants access to all six bundles at the ɳSelf+ price.

Bundle membership is authoritative at SPORT F06. Use `nself bundle list` for the live, CLI-computed list.

`nself bundle install sentry` is a slug alias for `nsentry` (added v1.2.3): it resolves to the same ɳSentry bundle but never appears in `nself bundle list` or shell completions — those always show the canonical `nsentry` slug.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `list` | Show all bundles, their plugins, and installation status |
| `info <bundle>` | Full detail for one bundle: plugins, pricing, requirements |
| `install <bundle>` | Install all plugins in a bundle (requires valid license) |
| `remove <bundle>` | Uninstall all plugins in a bundle (stops running plugin services) |

## Flags

### `bundle list`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | bool | false | Emit bundle list as JSON |
| `--installed` | bool | false | Show only installed bundles |

### `bundle info`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | bool | false | Emit info as JSON |

### `bundle install`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--key <key>` | string | — | License key to use (overrides `~/.nself/license/key`) |
| `--dry-run` | bool | false | Show what would be installed without installing |

### `bundle remove`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--yes` | bool | false | Skip confirmation |
| `--keep-data` | bool | false | Stop services but preserve plugin data volumes |

## Examples

```bash
# List all available bundles
nself bundle list
```

```bash
# See which plugins are in the ɳClaw bundle
nself bundle info nclaw
```

```bash
# Install ɳClaw (requires nclaw or nself-plus license)
nself bundle install nclaw
```

```bash
# `sentry` is a slug alias for `nsentry` (v1.2.3+) — both install the same bundle
nself bundle install sentry
```

```bash
# Preview what would be removed without removing
nself bundle remove nchat --dry-run
```

```bash
# Remove the ɳTV bundle and keep data volumes
nself bundle remove ntv --keep-data
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (license invalid, network failure, plugin error) |
| 2 | Unknown bundle name |

## Idempotency

`bundle install` is idempotent: running it twice produces no error if all plugins are already installed at the correct version. `bundle remove` is also idempotent: removing an already-removed bundle returns exit code 0.

## See Also

- [[cmd-license.md]] — manage license keys
- [[cmd-plugin.md]] — install individual plugins
- [[Licensing-Guide.md]] — bundle pricing and tiers
- [[Plugin-Overview.md]] — full plugin catalog

← [[Commands]] | [[Home]] →
