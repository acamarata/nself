# nself plugin dev

> Start a plugin in development mode with hot-reload.

## Synopsis

```bash
nself plugin dev <name> [flags]
```

## Description

`nself plugin dev` starts a named plugin in development mode, watching the plugin source directory for changes and reloading the process automatically. This is the primary workflow for plugin authors iterating on a plugin locally.

By default, the command auto-links the plugin before starting dev mode so the local source overrides the registry version. Passing `--no-link` skips this step if you have already linked manually.

Use `--debug` to attach a `dlv` debugger to the running plugin process. This delegates to `nself plugin debug` internally.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--no-link` | `false` | Skip auto-linking the plugin before starting dev mode |
| `--debug` | `false` | Attach dlv debugger (delegates to `nself plugin debug`) |
| `--entrypoint` | `./cmd` | Plugin entrypoint directory passed to dev-watch.sh |

## Examples

```bash
# Start the ai plugin in dev mode with hot-reload
nself plugin dev ai

# Start without auto-linking (assumes already linked)
nself plugin dev ai --no-link

# Start with debugger attached
nself plugin dev ai --debug

# Custom entrypoint
nself plugin dev ai --entrypoint ./cmd/main
```

## See Also

- [[cmd-plugin-link]] — Register a local plugin directory
- [[cmd-plugin-debug]] — Attach a debugger to a running plugin
- [[cmd-plugin-test]] — Run a plugin's test suite
- [[cmd-plugin-logs]] — Tail plugin container logs

← [[Commands]] | [[Home]] →
