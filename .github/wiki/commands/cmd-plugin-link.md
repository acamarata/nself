# nself plugin link / unlink

Register or remove a local plugin directory as a shadow override for development.

## Synopsis

```bash
nself plugin link <local-path> [flags]
nself plugin link --list
nself plugin unlink <name>
```

## Description

`plugin link` registers a local directory as a plugin shadow. On the next `nself build`,
the CLI uses your local source instead of the registry version. The link persists in
`~/.nself/plugin-links.json` across CLI invocations until you `unlink`.

`plugin unlink` removes the shadow and restores the registry version on the next build.

## Modes

| Mode | Flag | Description |
|------|------|-------------|
| Container-mount | (default) | Mounts source into container, runs air inside (closer to production) |
| Host-process | `--host` | Runs plugin process on host (faster I/O on macOS) |

The container-mount default works well on Linux and Windows WSL2. On macOS, `--host` is
recommended to avoid Docker's osxfs file-watching latency.

## Flags

### plugin link

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | false | Use host-process mode instead of container-mount |
| `--list` | false | List currently linked plugins and exit |

### plugin unlink

No additional flags. Unlinking an already-unlinked plugin is a no-op.

## Examples

```bash
# Link a plugin directory
nself plugin link /path/to/myplugin

# Link using host-process mode (recommended on macOS)
nself plugin link /path/to/myplugin --host

# List all currently linked plugins
nself plugin link --list

# Unlink a plugin (restores registry version on next build)
nself plugin unlink myplugin
```

## Validation

`plugin link` rejects paths that:

- Do not exist or are not directories
- Do not contain a `plugin.yaml` manifest

```
error: path "/tmp/notaplugin" does not contain plugin.yaml — not a valid plugin directory
```

## State File

Links are stored in `~/.nself/plugin-links.json` with `0600` permissions:

```json
{
  "myplugin": "/home/user/dev/myplugin",
  "another": "/home/user/dev/another"
}
```

## See Also

- [[cmd-plugin-dev]] — start hot-reload watcher
- [[cmd-plugin-test]] — run plugin test suite
- [[Home]]
