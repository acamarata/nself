# nself flag

Manage feature flags via the nself feature-flags plugin.

Feature flags let you toggle functionality, run canary rollouts, and kill-switch bad code paths
without a redeploy. All subcommands route through nginx. Port 3305 is never accessed directly.

## Usage

```bash
nself flag <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
|---|---|
| `list` | List all feature flags (filter by type) |
| `get <key>` | Get a single flag by key |
| `set <key>` | Update enabled state and/or rollout percentage |
| `enable <key>` | Enable a flag (sets enabled=true) |
| `disable <key>` | Disable a flag (sets enabled=false + pubsub broadcast) |
| `kill <key>` | Emergency kill-switch (requires --reason) |
| `history <key>` | Show audit log for a flag |
| `prune` | List/delete stale flags (past stale_after_days threshold) |

## Flag Types

| Type | Purpose |
|---|---|
| `release` | New feature rollout — percentage-based canary |
| `ops` | Operational toggle (rate limits, cache, circuit breakers) |
| `experiment` | A/B test variant |
| `kill_switch` | Emergency disable — never auto-enables |

## Rule Types (Evaluation)

| Type | Description |
|---|---|
| `percentage` | Random bucketing by user ID hash (0-100) |
| `user_id` | Exact UID allowlist |
| `group` | Named segment membership |
| `attribute` | Arbitrary context attribute match |
| `datetime` | Time-window gate (starts_at / ends_at) |

## Examples

```bash
# List all flags
nself flag list

# Filter by type
nself flag list --type release
nself flag list --type kill_switch --json

# Get a flag
nself flag get ai.safety.jailbreak_filter

# Enable / disable
nself flag enable ai.safety.jailbreak_filter
nself flag disable ai.safety.jailbreak_filter

# Set rollout percentage
nself flag set ai.safety.jailbreak_filter --rollout-pct 25

# Set enabled + rollout together
nself flag set ai.safety.jailbreak_filter --enabled --rollout-pct 50

# Kill-switch (--reason required)
nself flag kill ai.safety.jailbreak_filter --reason "CVE-2026-1234 mitigation"

# Audit log
nself flag history ai.safety.jailbreak_filter
nself flag history ai.safety.jailbreak_filter --json

# Stale flag scan
nself flag prune --stale --dry-run
nself flag prune --stale
```

## Flag Naming Convention

Format: `<plugin>.<category>.<name>`

Examples:
- `ai.safety.jailbreak_filter`
- `core.ratelimit.v2_algorithm`
- `claw.search.semantic_enabled`

CI enforces the convention via `scripts/scan-feature-flags.sh`.

## Kill vs Disable

| | `disable` | `kill` |
|---|---|---|
| Sets enabled=false | Yes | Yes |
| Requires --reason | No | **Yes** |
| Pubsub broadcast | Yes | Yes |
| Audit row | Yes | Yes |
| Intended use | Routine toggle | Emergency stop |

`kill` is the emergency path. Use `disable` for routine toggling.

## Flags (nself flag list)

| Flag | Description |
|---|---|
| `--type <type>` | Filter by flag type (release, ops, experiment, kill_switch) |
| `--json` | Output as JSON |

## Flags (nself flag set)

| Flag | Description |
|---|---|
| `--enabled` | Set enabled=true |
| `--no-enabled` | Set enabled=false |
| `--rollout-pct <n>` | Set rollout percentage (0-100) |

At least one of `--enabled` or `--rollout-pct` is required.

## Flags (nself flag kill)

| Flag | Description |
|---|---|
| `--reason <string>` | Reason for kill-switch (required, non-empty) |

## Prerequisite

The feature-flags plugin must be installed and running:

```bash
nself plugin install feature-flags
nself start
```

Check status:

```bash
nself status
```

## Related

- [[cmd-license]] — License management (triggers AI cost budget seeding)
- [[Plugin-Overview]] — All free plugins
- [Feature Flags Ops Doc](../.claude/docs/operations/feature-flags.md)

[[Home]]
