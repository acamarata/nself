# nself plugin unlink

> Remove a plugin from the development-linked set.

## Synopsis

```
nself plugin unlink <name> [flags]
```

## Description

`nself plugin unlink` removes a previously linked plugin from the running nSelf stack. After unlinking, the plugin is no longer visible to dependent services. Run `nself build` and `nself restart` to restore the stack to its pre-link state.

This command is the counterpart to `nself plugin link`. Use it when you finish local development on a plugin and want to return to the installed version.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Unlink a plugin by name
nself plugin unlink my-plugin
```

```bash
# Unlink and rebuild the stack
nself plugin unlink my-plugin && nself build && nself restart
```

## See Also

- [[cmd-plugin-link]] — link a local plugin directory into the stack
- [[cmd-plugin-dev]] — start dev mode with live reload
- [[cmd-plugin]] — plugin command overview
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
