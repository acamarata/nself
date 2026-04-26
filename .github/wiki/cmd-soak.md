# nself soak

> Manage soak testing lifecycle for ɳSelf environments.

## Synopsis

```
nself soak <subcommand>
nself soak abort --rollback <version> [flags]
```

## Description

`nself soak` provides lifecycle management for soak tests. The `abort` subcommand
lets you stop an active soak and roll back to a prior version in a single command,
replacing the previous PCI + 5-minute human workflow.

## Subcommands

| Subcommand | Description |
|---|---|
| `abort` | Abort a running soak and roll back to a prior version |

---

## nself soak abort

Abort an active soak on a target environment and roll back to the specified version.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--rollback <version>` | — | **Required.** Version to roll back to |
| `--dry-run` | false | Print exact rollback steps without executing |
| `--yes` | false | Skip the interactive confirmation prompt |
| `--env` | `staging` | Target environment: `local`, `staging`, `prod`/`production` |
| `--prod-i-mean-it` | false | Required when `--env prod` is specified |

### Behavior

1. Validates environment gate, production requires `--prod-i-mean-it`
2. Checks idempotency, if the environment is already at the target version, exits with a clear message (no-op)
3. Prompts for interactive confirmation unless `--yes` is passed
4. Tags current state with a timestamp for post-mortem (`pre-abort-<env>-<ts>`)
5. Executes deploy rollback to the target version
6. Posts a notify event with the rollback reason

Per the RISK doctrine, this command never executes without explicit confirmation
unless `--yes` is passed.

### Examples

```bash
# Preview what would happen (no changes)
nself soak abort --rollback v1.0.8 --dry-run

# Abort staging soak with interactive confirmation
nself soak abort --rollback v1.0.8

# Abort staging soak without confirmation prompt
nself soak abort --rollback v1.0.8 --yes

# Abort production soak (requires --prod-i-mean-it)
nself soak abort --rollback v1.0.8 --env prod --prod-i-mean-it --yes
```

### Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success or idempotent no-op |
| 1 | Rollback failed , see error output for details |

### See Also

- [nself deploy rollback](cmd-deploy.md), underlying rollback mechanism
- [Observability](https://docs.nself.org/operations/observability), post-rollback metrics
