# nself plugin debug

Attach a Delve debugger to a plugin process for step-through debugging.

## Synopsis

```bash
nself plugin debug <name> [flags]
```

## Description

Builds the named plugin with debug symbols disabled (`-gcflags=all=-N -l`), then starts
it under `dlv exec --headless`. Auto-allocates a port from the 2345-2399 range and prints
a VS Code `launch.json` snippet you can paste directly.

Multiple plugins can be debugged simultaneously — each gets its own port from the range.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | 0 (auto) | Debugger listen port (auto-allocates from 2345-2399 if 0) |
| `--port-only` | false | Print the allocated port and exit (for scripting) |

## Examples

```bash
# Start debugger on auto-allocated port
nself plugin debug myplugin

# Use a specific port
nself plugin debug myplugin --port 2350

# Print port for scripting
PORT=$(nself plugin debug myplugin --port-only)
echo "Connect on :$PORT"

# Combined with nself plugin dev (in two terminals)
# Terminal 1:
nself plugin dev myplugin --no-link --debug
# Terminal 2 (VS Code attach or dlv connect):
dlv connect :2345
```

## Output

```
Building myplugin for debug...
Plugin built at /tmp/nself-debug-myplugin
Starting dlv on :2345...
--- VS Code launch.json snippet ---
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Attach to myplugin",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "remotePath": "${workspaceFolder}",
      "port": 2345,
      "host": "127.0.0.1"
    }
  ]
}
---
GoLand: Run > Attach to Process > Remote... > host=127.0.0.1, port=2345
---
```

## Prerequisites

Install Delve:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

## Port Allocation

Ports 2345-2399 are the debug port range (55 simultaneous sessions). If the range is
exhausted, use `--port` to specify a port outside the range.

The port is also exposed via `--port-only` for automation:

```bash
nself plugin debug myplugin --port-only   # prints :2345
```

## Security Note

`--accept-multiclient` is a development-only flag and is NOT enabled in production builds.
The debugger only listens on `127.0.0.1` — never exposed to external networks.

## See Also

- [[cmd-plugin-dev]] — hot-reload watcher (includes `--debug` shortcut)
- [[cmd-plugin-test]] — automated test suite
- [[Home]]
