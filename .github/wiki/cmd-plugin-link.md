# nself plugin link

> Link a local plugin directory into the running nSelf stack for development.

## Synopsis

```
nself plugin link <local-path> [flags]
```

## Description

`nself plugin link` mounts a local plugin source directory into the running nSelf stack so the in-development plugin is visible to dependent services without a full install. This is the first step in the plugin author workflow: link, then use `nself plugin dev` or `nself plugin test` to iterate.

By default, linking uses container-mount mode, which requires Docker volume mounts. On macOS, `--host` enables host-process mode for faster I/O by bypassing Docker's filesystem virtualization layer.

Use `--list` to see all currently linked plugins without linking a new one.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | false | Use host-process mode instead of container-mount (recommended on macOS for faster I/O) |
| `--list` | false | List currently linked plugins and exit |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Link a plugin from the current directory
nself plugin link .
```

```bash
# Link a plugin from an explicit path
nself plugin link ./plugins/my-plugin
```

```bash
# Link using host-process mode (faster on macOS)
nself plugin link . --host
```

```bash
# List all currently linked plugins
nself plugin link --list
```

## See Also

- [[cmd-plugin-unlink]] — remove a plugin from the linked set
- [[cmd-plugin-dev]] — start dev mode with live reload
- [[cmd-plugin-test]] — run unit and smoke tests
- [[cmd-plugin]] — plugin command overview
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
