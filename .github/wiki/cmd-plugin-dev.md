# nself plugin dev

> Start a plugin in development mode with live reload.

## Synopsis

```
nself plugin dev <name> [flags]
```

## Description

`nself plugin dev` starts a named plugin in development mode with automatic hot-reload on source changes. It uses `air` if installed, falling back to `fswatch` polling.

Before starting, the command auto-links the plugin into the running nSelf stack so changes are immediately visible to dependent services. Use `--no-link` to skip linking when you have already called `nself plugin link` manually.

Pass `--debug` to delegate to `nself plugin debug`, which attaches a Delve debugger instead of a reload watcher. The `--entrypoint` flag sets the plugin's Go entrypoint directory (default: `./cmd`), used by `dev-watch.sh`.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--no-link` | false | Skip auto-linking the plugin before starting dev mode |
| `--debug` | false | Attach a dlv debugger instead of the reload watcher (delegates to `plugin debug`) |
| `--entrypoint` | `./cmd` | Plugin entrypoint directory passed to `dev-watch.sh` |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Start dev mode for a linked plugin
nself plugin dev my-plugin
```

```bash
# Start dev mode without auto-linking (manual link already done)
nself plugin dev my-plugin --no-link
```

```bash
# Start dev mode with a non-default entrypoint
nself plugin dev my-plugin --entrypoint ./server
```

```bash
# Start dev mode with the Delve debugger attached
nself plugin dev my-plugin --debug
```

## See Also

- [[cmd-plugin-link]] — link a local plugin directory into the stack
- [[cmd-plugin-debug]] — attach a Delve debugger to a running plugin
- [[cmd-plugin-test]] — run plugin unit and smoke tests
- [[cmd-plugin]] — plugin command overview
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
