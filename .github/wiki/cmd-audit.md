# nself audit

> Run ecosystem audits: documentation coverage, origin consistency, and quarterly review gates.

## Synopsis

```
nself audit <subcommand> [flags]
```

## Description

`nself audit` runs structured audits across the nSelf ecosystem. Each subcommand targets a specific audit surface.

The `docs` subcommand checks documentation coverage against code reality: command pages against registered cobra commands, wiki pages against plugin registry, env var references against `.env.example`. It is the same check the CI doc-sync gate runs on every PR. Running it locally surfaces drift before it reaches CI.

## Subcommands

| Name | Description |
|------|-------------|
| `docs` | Run quarterly documentation audit: commands vs wiki, plugins vs registry, env vars vs `.env.example` |

## Flags

### `audit docs`

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `table` | Output format: `table` or `json` |
| `--fail-on-drift` | false | Exit non-zero when any drift is found (default in CI) |
| `--only` | `""` | Run only one surface: `commands`, `plugins`, `env` |

## Examples

```bash
# Run full documentation audit (table output)
nself audit docs

# Check only command wiki coverage
nself audit docs --only commands

# JSON output for CI integration
nself audit docs --format json --fail-on-drift
```

## Notes

- `nself audit docs` is invoked automatically by the `doc-sync.yml` CI workflow on every PR that touches `cmd/commands/` or `.github/wiki/`.
- Drift findings reference the canonical SPORT files at `~/Sites/nself/.claude/docs/sport/F02-COMMAND-INVENTORY.md` and `F03-PLUGIN-INVENTORY-FREE.md`.
- The `--fail-on-drift` flag is set by default in CI. Running locally without it produces a report but always exits 0.

## See Also

- [[cmd-doctor]] — system diagnostics
- [[cmd-security]] — security audit
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
