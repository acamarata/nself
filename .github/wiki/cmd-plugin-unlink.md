# nself plugin unlink

> Remove a local plugin shadow, restoring the registry version.

## Synopsis

```bash
nself plugin unlink <name>
```

## Description

`nself plugin unlink` removes the local shadow override for a plugin and restores the installed registry version. Use this after finishing local development to return to the published plugin.

This is the counterpart to `nself plugin link`. After unlinking, `nself plugin status <name>` will show the registry version is active again.

## Flags

No flags.

## Examples

```bash
# Unlink the ai plugin and restore registry version
nself plugin unlink ai

# Unlink a custom plugin
nself plugin unlink my-plugin
```

## See Also

- [[cmd-plugin-link]] — Register a local plugin directory as a shadow override
- [[cmd-plugin-dev]] — Start a plugin in development mode
- [[cmd-plugin-status]] — Show status of a plugin

← [[Commands]] | [[Home]] →
