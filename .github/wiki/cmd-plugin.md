# nself plugin

> Install, remove, update, and manage nSelf plugins.

## Synopsis

```
nself plugin <subcommand> [flags]
```

## Description

`nself plugin` manages the nSelf plugin ecosystem. Plugins extend the CLI and your backend stack with new capabilities. Free plugins (MIT licensed) install without a key. Pro plugins require a valid membership license key — set one with `nself license set`.

When you install a plugin, nSelf checks your license tier against the plugin's requirements, downloads the plugin binary and Docker image, registers the plugin with the stack, and prepares database migrations. Run `nself build` and `nself restart` after installing plugins to include them in the generated `docker-compose.yml`.

Unknown subcommands are proxied to the matching plugin binary: `nself plugin ai <action>` calls `nself-ai <action>`. This allows installed plugins to expose their own subcommands through the nSelf CLI namespace.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `install <plugin> [plugin...]` | Install one or more plugins (license check enforced for pro plugins; planned plugins are rejected) |
| `remove <name>` | Remove a plugin |
| `update [name]` | Update a specific plugin, or all installed plugins if no name given |
| `updates` | Check for available updates across all installed plugins |
| `list` | List available and installed plugins (beta and planned plugins show status badges) |
| `compat-check` | Check installed plugins for CLI version compatibility (exits 1 on any incompatible plugin) |
| `inventory` | List installed plugins with version, tier, and status |
| `refresh` | Force refresh the remote registry cache |
| `start <name>` | Start a plugin service container |
| `stop <name>` | Stop a plugin service container |
| `status [name]` | Show plugin health (all or specific) |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--key` | `""` | License key for pro plugins (`install`) |
| `--version` | `""` | Install a specific version (`install`) |
| `--force` | false | `install`: required when `NSELF_LICENSE_SKIP_VERIFY=1` is set; `remove`: remove even if dependents exist |
| `--keep-data` | false | Preserve database data when removing |
| `--installed` | false | Show only installed plugins (for `list`) |
| `--show-eol` | false | Include end-of-life plugins in `list` output (hidden by default) |
| `--allow-eol` | false | Allow installing or updating an EOL plugin (not recommended) |
| `--category` | `""` | Filter by category (for `list`) |
| `--detailed` | false | Show detailed information |
| `--help`, `-h` | — | Show help |

## Plugin Status Badges

Plugins carry a status field in the registry. The `list` subcommand shows badges for non-stable plugins:

```
ai                  [installed]
browser             [beta]
nfamily             [planned]
```

Install behavior by status:

- **stable** — installs without warnings (default for most plugins)
- **experimental** — prints a warning to stderr, then installs
- **beta** — prints a warning to stderr, then installs
- **deprecated** — prints a deprecation warning with EOL date and migration guide, then installs
- **eol** — install is blocked; use `--allow-eol` to override (not recommended)
- **planned** — install is rejected with a "coming soon" message and a link to the release timeline

EOL plugins are hidden from `nself plugin list` by default. Use `--show-eol` to include them.

See [[Plugin-Status-Badges]] for the full reference.

## Examples

```bash
# List all available plugins
nself plugin list

# List only installed plugins
nself plugin list --installed

# Filter by category
nself plugin list --category ai

# Install a free plugin
nself plugin install notify

# Install multiple plugins in one command
nself plugin install ai claw mux

# Install a pro plugin (uses saved license key)
nself plugin install ai

# Install a pro plugin with an inline key
nself plugin install livekit --key nself_pro_xxxxx...

# Install a specific version
nself plugin install recording --version 1.2.0

# Remove a plugin
nself plugin remove ai

# Remove a plugin but keep its database data
nself plugin remove livekit --keep-data

# Force remove (ignores dependents)
nself plugin remove recording --force

# Update a specific plugin
nself plugin update ai

# Update all plugins
nself plugin update

# Check for available updates without installing
nself plugin updates

# Show installed plugins with version and tier
nself plugin inventory

# Refresh registry cache
nself plugin refresh

# Start/stop a plugin service
nself plugin start ai
nself plugin stop ai

# Show plugin status
nself plugin status
nself plugin status ai --detailed

# Check compatibility after a CLI upgrade
nself plugin compat-check

# List all plugins including EOL ones
nself plugin list --show-eol

# Install an EOL plugin (not recommended — use only when a replacement is unavailable)
nself plugin install old-plugin --allow-eol
```

## See also

- [[cmd-plugin-compat-check]] — compatibility check reference
- [[Plugin-Status-Badges]] — lifecycle status reference
- [[Plugin-Licensing]] — license tiers and key format

← [[Commands]] | [[Home]] →
