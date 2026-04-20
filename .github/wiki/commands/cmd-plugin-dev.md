# nself plugin dev

Start a plugin in development mode with hot-reload.

## Synopsis

```bash
nself plugin dev <name> [flags]
```

## Description

Wraps the plugin SDK's `dev-watch.sh` script to give plugin authors a first-class
inner development loop. On first run, auto-links the plugin directory into the nSelf
shadow registry so changes are picked up immediately on the next `nself build`.

Prefers `air` for hot-reload. Falls back to `fswatch` polling if `air` is not installed.
Falls back to a bundled inline watch script if neither is available.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--no-link` | false | Skip auto-link step (manage linking manually) |
| `--debug` | false | Attach dlv debugger (delegates to `nself plugin debug`) |
| `--entrypoint` | `./cmd` | Plugin entrypoint directory passed to dev-watch.sh |

## Examples

```bash
# Start hot-reload watcher with auto-link
nself plugin dev myplugin

# Skip auto-link (plugin already linked)
nself plugin dev myplugin --no-link

# Start with debugger attached
nself plugin dev myplugin --debug

# Custom entrypoint
nself plugin dev myplugin --entrypoint ./cmd/server
```

## Output

```
Starting myplugin in development mode...
Plugin path: /home/user/plugins/myplugin
Auto-linking myplugin...
Plugin linked.
Using watch script: /usr/local/share/plugin-sdk-go/devkit/tools/dev-watch.sh
Entrypoint: ./cmd
Watching for changes — press Ctrl+C to stop.
```

## Prerequisites

Install `air` for the best hot-reload experience:

```bash
go install github.com/air-verse/air@latest
```

Or install `fswatch` (macOS):

```bash
brew install fswatch
```

## See Also

- [[cmd-plugin-link]] — link a local plugin directory
- [[cmd-plugin-debug]] — attach dlv debugger
- [[cmd-plugin-test]] — run plugin test suite
- [[cmd-plugin-logs]] — tail plugin container logs
- [[Home]]
