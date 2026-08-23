# nself dogfood

<!-- BEGIN PROSE:summary -->
> Production dogfood audit and reporting.
<!-- END PROSE:summary -->

## Synopsis

```
nself dogfood <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself dogfood` runs the dogfood audit checklist against a production environment and shows the latest report. The audit covers backups, disaster recovery, multi-tenancy, licensing, secrets, migrations, monitoring, security posture, watchdog state, and queue health: 21 checks grouped by section.

`dogfood audit` runs the full check pass and exits with a status code that machines can act on: `0` when every check passes, `1` when any check fails, `2` when there are warnings only. By default it saves the report to `.nself/dogfood/` so subsequent `dogfood report` invocations can replay the latest snapshot without re-running checks.

Use this command before promoting a release, after major infrastructure changes, and on a recurring schedule (e.g. weekly cron) to detect drift. JSON output is suitable for piping into dashboards.

### `dogfood audit`
### `dogfood report`
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
| `audit` | Run comprehensive dogfood audit (21 checks) |
| `report` | Show the latest dogfood audit report |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Run the full audit and save the report
nself dogfood audit

# Run audit and emit JSON for a dashboard
nself dogfood audit --json > dogfood-latest.json

# Show only the failures and warnings from the latest report
nself dogfood report

# Re-print the latest report as JSON
nself dogfood report --json
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-doctor]], system diagnostics
- [[cmd-security]], security audit
- [[cmd-watchdog]], self-healing watchdog
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
