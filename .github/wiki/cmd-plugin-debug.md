# nself plugin debug

> Attach a Delve debugger to a running plugin process.

## Synopsis

```
nself plugin debug <name> [flags]
```

## Description

`nself plugin debug` builds the named plugin with debug symbols (`-gcflags=all=-N -l`) and starts it under the Delve debugger in headless mode. It auto-allocates a port from the `2345–2399` range, handling simultaneous debug sessions for multiple plugins without port conflicts.

After starting, the command prints the allocated port and a VS Code `launch.json` snippet you can paste into your editor to connect. Connect any Delve-compatible debugger (VS Code Go extension, GoLand, or `dlv connect`) to the printed address.

Requires `dlv` to be installed: `go install github.com/go-delve/delve/cmd/dlv@latest`.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `0` | Debugger listen port (0 = auto-allocate from 2345–2399) |
| `--port-only` | false | Print the allocated port and exit without starting the debugger (for scripting) |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Start the debugger on an auto-allocated port
nself plugin debug my-plugin
```

```bash
# Use a fixed port
nself plugin debug my-plugin --port 2350
```

```bash
# Get just the port number (for scripting)
nself plugin debug my-plugin --port-only
```

## See Also

- [[cmd-plugin-dev]] — start dev mode with live reload (use `--debug` to delegate here)
- [[cmd-plugin-test]] — run unit and smoke tests
- [[cmd-plugin-logs]] — tail plugin container logs
- [[cmd-plugin]] — plugin command overview
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
