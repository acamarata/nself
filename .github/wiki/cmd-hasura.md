# nself hasura

<!-- BEGIN PROSE:summary -->
> Hasura metadata operations (alias for 'nself db hasura').
<!-- END PROSE:summary -->

## Synopsis

```
nself hasura <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
Shorter top-level aliases for the most-used `nself db hasura` metadata operations. Every subcommand here delegates to the exact same implementation as its `nself db hasura ...` counterpart — there is no behavior difference, only a shorter path to type.

As of P6, `hasura/metadata/` (or the legacy `hasura/metadata.json`) is also applied automatically by `nself start` and `nself deploy`, so most projects will only reach for these commands to inspect drift (`diff`) or re-apply by hand after editing metadata directly. See [[operations/hasura-metadata-auto-apply]] for the full auto-apply behavior, strict/warn-only rules, and remote (`--env staging|prod`) targeting.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `diff` | Compare live Hasura metadata against on-disk files (alias for 'nself db hasura diff') |
| `metadata` | Manage Hasura metadata |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Re-apply hasura/metadata/ to the local running stack by hand
nself hasura metadata apply

# Same, against staging or prod over SSH (see 'nself db hasura --help' for --server)
nself hasura metadata apply --env staging

# Export the live metadata back to git-friendly sorted YAML
nself hasura metadata export

# Check for drift between on-disk metadata and what Hasura is actually tracking
nself hasura diff
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[operations/hasura-metadata-auto-apply]] — auto-apply on start/deploy, strict vs. warn-only
- [[operations/hasura-metadata-backup]] — daily metadata export + restore
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
