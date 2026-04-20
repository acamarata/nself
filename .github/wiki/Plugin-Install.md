# Plugin Install

## Install a Plugin

```bash
nself plugin install {name}
```

For free plugins, no setup required. For pro plugins, set your license key first — see [[Plugin-Licensing]].

Plugins marked `beta` install with a warning printed to stderr. Plugins marked `planned` are not yet available — the install command returns an error with a link to the release timeline. See [[Plugin-Status-Badges]] for details.

After installing any plugin, regenerate your stack:

```bash
nself build
nself restart
```

## Remove a Plugin

```bash
nself plugin remove {name}
```

By default, the plugin's database schema (`np_{name}`) is dropped. To keep data:

```bash
nself plugin remove {name} --keep-data
```

## Update Plugins

```bash
# Update a single plugin
nself plugin update {name}

# Update all installed plugins
nself plugin update --all
```

## List Installed Plugins

```bash
nself plugin list
```

Shows all installed plugins with version and status.

## Get Plugin Info

```bash
nself plugin info {name}
```

Shows plugin description, version, port, tables, dependencies, and env vars.

## Check Available Plugins

```bash
# Browse the free registry
nself plugin search

# Search by keyword
nself plugin search {keyword}
```

## Install Multiple Plugins

```bash
nself plugin install chat livekit streaming
nself build
```

## Plugin Dependencies

Some plugins require others to be installed first. nSelf resolves dependencies automatically during install. If a required plugin isn't installed, nSelf installs it for you.

Example: `claw` requires `ai`. Installing `claw` automatically installs `ai` if missing.

## Environment Variables

Each plugin declares its required env vars in `plugin.json`. After install, add the required vars to your `.env` file:

```bash
nself plugin info {name}   # shows required env vars
```

Then run `nself build` to pick up the new configuration.

## Related Pages

- [[Plugin-Overview]] — What plugins are and pricing tiers
- [[Plugin-Licensing]] — License keys for pro plugins
- [[Plugin-Status-Badges]] — stable, beta, and planned behavior at install time
- [[Plugin-Dev-Guide]] — Build your own plugin
