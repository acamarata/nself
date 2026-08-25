# nself flags

<!-- BEGIN PROSE:summary -->
> Manage application feature flags.
<!-- END PROSE:summary -->

## Synopsis

```
nself flags <subcommand> [flags]
```

**Alias:** `nself flag`

## Description

<!-- BEGIN PROSE:description -->
Manage feature flags via the nself feature-flags plugin.

Feature flags let you toggle functionality, run canary rollouts, and kill-switch
bad code paths without a redeploy. All operations route through nginx (port 3305 is
never accessed directly).

Flag types:
  release      New feature rollout (percentage-based)
  ops          Operational toggle (rate limits, cache tuning, etc.)
  experiment   A/B test variant
  kill_switch  Emergency disable — never auto-enables

Rule types (for evaluation):
  percentage   Random bucketing by user ID hash (0-100)
  user_id      Exact UID allowlist
  group        Named segment membership
  attribute    Arbitrary context attribute match
  datetime     Time-window gate (starts_at / ends_at)
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
| `disable` | Disable a feature flag |
| `enable` | Enable a feature flag |
| `get` | Get a single feature flag |
| `history` | Show audit log for a feature flag |
| `kill` | Kill-switch a feature flag (emergency disable with required reason) |
| `list` | List all feature flags |
| `prune` | List (or delete) stale feature flags |
| `set` | Update a feature flag's enabled state and/or rollout percentage |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
<!-- TODO(docs): needs human prose -->

```bash
nself flags
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
