# nself remove

<!-- BEGIN PROSE:summary -->
> Remove an installed plugin or bundle.
<!-- END PROSE:summary -->

## Synopsis

```
nself remove <name> [flags]
```

**Alias:** `nself rm`

## Description

<!-- BEGIN PROSE:description -->
Remove an installed plugin or bundle by name.

This is the short form of `nself plugin remove`. As with install, a bundle
name removes the whole bundle.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--yes` | `false` | Skip confirmation prompts |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
nself remove waf
  nself remove nchat
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
