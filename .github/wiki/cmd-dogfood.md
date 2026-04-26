# nself dogfood

> Production dogfood audit and reporting.

## Synopsis

```
nself dogfood <subcommand> [flags]
```

## Description

`nself dogfood` runs the dogfood audit checklist against a production environment and shows the latest report. The audit covers backups, disaster recovery, multi-tenancy, licensing, secrets, migrations, monitoring, security posture, watchdog state, and queue health: 21 checks grouped by section.

`dogfood audit` runs the full check pass and exits with a status code that machines can act on: `0` when every check passes, `1` when any check fails, `2` when there are warnings only. By default it saves the report to `.nself/dogfood/` so subsequent `dogfood report` invocations can replay the latest snapshot without re-running checks.

Use this command before promoting a release, after major infrastructure changes, and on a recurring schedule (e.g. weekly cron) to detect drift. JSON output is suitable for piping into dashboards.

## Subcommands

| Name | Description |
|------|-------------|
| `audit` | Run the dogfood audit (21 checks) |
| `report` | Show the latest dogfood audit report |

## Flags

### `dogfood audit`

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | JSON output |
| `--save` | true | Save report to `.nself/dogfood/` |

### `dogfood report`

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | JSON output |

## Examples

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

## See Also

- [[cmd-doctor]], system diagnostics
- [[cmd-security]], security audit
- [[cmd-watchdog]], self-healing watchdog
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
