# nself plugin link

> Register a local plugin directory as a shadow override.

## Synopsis

```bash
nself plugin link <local-path> [flags]
nself plugin unlink <name> [flags]
```

## Description

`nself plugin link` registers a local directory as a shadow override for a plugin. When a plugin is linked, nSelf loads it from the local path instead of the installed registry version. This is the standard workflow for plugin authors testing changes without publishing.

`nself plugin unlink` removes the local shadow and restores the registry version.

Use `--list` to see all currently linked plugins. Use `--host` to mount the plugin in host-process mode rather than a container — recommended on macOS for faster I/O.

## Flags

### link

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `false` | Use host-process mode instead of container-mount (macOS recommended for faster I/O) |
| `--list` | `false` | List currently linked plugins |

### unlink

No additional flags.

## Examples

```bash
# Link a local plugin directory
nself plugin link ./my-plugin

# Link in host-process mode (faster on macOS)
nself plugin link ./my-plugin --host

# List all currently linked plugins
nself plugin link --list

# Unlink a plugin (restores registry version)
nself plugin unlink ai
```

## See Also

- [[cmd-plugin-dev]] — Start a plugin in development mode
- [[Plugin-Install]] — Install a plugin from the registry
- [[cmd-plugin-test]] — Run a plugin's test suite

← [[Commands]] | [[Home]] →
