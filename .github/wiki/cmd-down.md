# nself down

> Alias for `nself stop`. Shuts down the ɳSelf stack.

## Synopsis

```
nself down [SERVICES...] [flags]
```

## Description

`nself down` is a hidden alias for [[cmd-stop]]. It is registered in `cmd/commands/aliases.go` and re-uses the same `RunE` handler as `stopCmd`. Flags, positional arguments, and behavior are identical.

Use either command interchangeably. `down` exists for muscle memory from `docker compose down`.

## Flags

See [[cmd-stop]] for the full flag list (includes `--graceful`, `--volumes`, `--rmi`, `--remove-orphans`). The alias passes through every flag without modification.

## Examples

```bash
# Same as: nself stop
nself down
```

```bash
# Stop only specific services
nself down postgres redis
```

```bash
# Remove volumes (destroys data)
nself down --volumes
```

## See Also

- [[cmd-stop]], canonical command (this page redirects to it)
- [[cmd-start]], boot the stack
- [[cmd-up]], alias for start
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
