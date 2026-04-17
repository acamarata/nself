# nself up

> Alias for `nself start`. Boots the nSelf stack.

## Synopsis

```
nself up [flags]
```

## Description

`nself up` is a hidden alias for [[cmd-start]]. It is registered in `cmd/commands/aliases.go` and re-uses the same `RunE` handler as `startCmd`. Flags and behavior are identical.

Use either command interchangeably. `up` exists for muscle memory from `docker compose up`.

## Flags

See [[cmd-start]] for the full flag list. The alias passes through every flag without modification.

## Examples

```bash
# Same as: nself start
nself up
```

```bash
# Alias honors all start flags
nself up --verbose
```

## See Also

- [[cmd-start]] — canonical command (this page redirects to it)
- [[cmd-stop]] — shut the stack down
- [[cmd-down]] — alias for stop
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
