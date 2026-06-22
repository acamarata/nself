# nself plugin

> Install, remove, update, and manage ɳSelf plugins.

## Synopsis

```
nself plugin <subcommand> [flags]
```

## Description

`nself plugin` manages the ɳSelf plugin ecosystem. Plugins extend the CLI and your backend stack with new capabilities. Free plugins (MIT licensed) install without a key. Pro plugins require a valid membership license key, set one with `nself license set`.

When you install a plugin, ɳSelf checks your license tier against the plugin's requirements, downloads the plugin binary and Docker image, registers the plugin with the stack, and prepares database migrations. Run `nself build` and `nself restart` after installing plugins to include them in the generated `docker-compose.yml`.

Unknown subcommands are proxied to the matching plugin binary: `nself plugin ai <action>` calls `nself-ai <action>`. This allows installed plugins to expose their own subcommands through the ɳSelf CLI namespace.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `list` | List available and installed plugins (beta and planned plugins show status badges) |
| `install <plugin> [plugin...]` | Install one or more plugins (license check enforced for pro plugins; planned plugins are rejected) |
| `remove <name>` | Remove a plugin and clean up its data, ports, and service entries |
| `update [name]` | Update a specific plugin, or all installed plugins if no name given |
| `updates` | Check for available updates across all installed plugins |
| `refresh` | Force refresh the remote registry cache |
| `start <name>` | Start a plugin service container |
| `stop <name>` | Stop a plugin service container |
| `disable <name>` | Disable a plugin (excluded from compose on next build) |
| `enable <name>` | Re-enable a previously disabled plugin |
| `status [name]` | Show plugin health (all or specific) |
| `inventory` | List installed plugins with version, tier, and status |
| `info <name>` | Show detailed information about a plugin |
| `new <name>` | Scaffold a new plugin project (preferred; see [[cmd-plugin-init]]) |
| `submit [path]` | Validate a plugin for submission (no actual upload in CI; use `--strict` for full checks) |
| `marketplace` | Browse the plugin marketplace (subcommands: `list`, `search`, `info`) |
| `compat-check` | Check installed plugins for CLI version compatibility (exits 1 on any incompatible plugin; see [[cmd-plugin-compat-check]]) |
| `dev <name>` | Start a plugin in development mode with live reload (see [[cmd-plugin-dev]]) |
| `link <local-path>` | Link a local plugin directory into the running stack (see [[cmd-plugin-link]]) |
| `unlink <name>` | Remove a plugin from the development-linked set (see [[cmd-plugin-unlink]]) |
| `test <name>` | Run unit and smoke tests for a plugin (see [[cmd-plugin-test]]) |
| `debug <name>` | Attach a Delve debugger to a running plugin process (see [[cmd-plugin-debug]]) |
| `logs <name>` | Tail logs from a plugin container (see [[cmd-plugin-logs]]) |

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

- **stable**, installs without warnings (default for most plugins)
- **experimental**, prints a warning to stderr, then installs
- **beta**, prints a warning to stderr, then installs
- **deprecated**, prints a deprecation warning with EOL date and migration guide, then installs
- **eol**, install is blocked; use `--allow-eol` to override (not recommended)
- **planned**, install is rejected with a "coming soon" message and a link to the release timeline

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

## Install telemetry

After a successful `nself plugin install`, the CLI sends a single fire-and-forget event to `plugins.nself.org/plugins/:name/install-event`. This increments the public download counter shown in the plugin marketplace.

The event body contains one field: `instanceId`, which is an opaque SHA-256 hash of a machine-local identifier. No hostname, IP address, username, or project name is transmitted. The event is deduplicated per (instance, plugin) per ISO week, so reinstalling the same plugin in the same week does not double-count. If the network is unavailable, the event is silently dropped with no retry.

To opt out, set `NSELF_DISABLE_TELEMETRY=1` in your environment or `.env.local`.

## See also

- [[cmd-plugin-compat-check]] — compatibility check reference
- [[cmd-plugin-dev]] — plugin author dev mode
- [[cmd-plugin-link]] — link a local plugin directory into the stack
- [[cmd-plugin-unlink]] — remove a plugin from the linked set
- [[cmd-plugin-test]] — run unit and smoke tests
- [[cmd-plugin-debug]] — attach a Delve debugger
- [[cmd-plugin-logs]] — tail plugin container logs
- [[cmd-plugin-marketplace]] — browse the marketplace
- [[Plugin-Status-Badges]] — lifecycle status reference
- [[Plugin-Licensing]] — license tiers and key format

← [[Commands]] | [[Home]] →
