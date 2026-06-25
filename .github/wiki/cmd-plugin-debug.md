# nself plugin debug

> Attach a dlv debugger to a running plugin process.

## Synopsis

```bash
nself plugin debug <name> [flags]
```

## Description

`nself plugin debug` attaches a `dlv` (Delve) debugger to a running plugin process. The plugin must already be running (via `nself plugin start` or `nself plugin dev`) before the debugger can attach.

By default, nSelf auto-allocates a debugger port from the range 2345-2399. Use `--port` to specify a fixed port. Use `--port-only` to print the allocated port without attaching — useful for scripting or connecting an external IDE.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `0` | Debugger listen port (0 = auto-allocate from 2345-2399) |
| `--port-only` | `false` | Print the allocated port and exit (for scripting) |

## Examples

```bash
# Attach debugger to the ai plugin (auto-allocate port)
nself plugin debug ai

# Attach on a fixed port
nself plugin debug ai --port 2345

# Print the allocated port for IDE configuration
nself plugin debug ai --port-only
```

## See Also

- [[cmd-plugin-dev]] — Start a plugin in development mode with hot-reload
- [[cmd-plugin-test]] — Run a plugin's test suite
- [[cmd-plugin-logs]] — Tail plugin container logs

← [[Commands]] | [[Home]] →
