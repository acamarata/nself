# nself audit

<!-- BEGIN PROSE:summary -->
> Run ecosystem audits: documentation coverage, origin consistency, and quarterly review gates.
<!-- END PROSE:summary -->

## Synopsis

```
nself audit <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself audit` runs structured audits across the ɳSelf ecosystem. Each subcommand targets a specific audit surface.

The `docs` subcommand checks documentation coverage against code reality: command pages against registered cobra commands, wiki pages against plugin registry, env var references against `.env.example`. It is the same check the CI doc-sync gate runs on every PR. Running it locally surfaces drift before it reaches CI.

### `audit docs`
## Notes

- `nself audit docs` is invoked automatically by the `doc-sync.yml` CI workflow on every PR that touches `cmd/commands/` or `.github/wiki/`.
- Drift findings reference the canonical SPORT files at `~/Sites/nself/.claude/docs/sport/F02-COMMAND-INVENTORY.md` and `F03-PLUGIN-INVENTORY-FREE.md`.
- The `--fail-on-drift` flag is set by default in CI. Running locally without it produces a report but always exits 0.
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
| `docs` | Run the quarterly documentation audit |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Run full documentation audit (table output)
nself audit docs

# Check only command wiki coverage
nself audit docs --only commands

# JSON output for CI integration
nself audit docs --format json --fail-on-drift
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-doctor]], system diagnostics
- [[cmd-security]], security audit
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
