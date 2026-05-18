# nself self-heal

> Run targeted repair routines for nSelf components without a full stack rebuild.

## Synopsis

```
nself self-heal [flags]
```

## Description

`nself self-heal` runs specific repair routines against the current nSelf project. Each routine is selected by a flag. You can combine multiple flags in one invocation to run several repairs in sequence.

With no flags, the command prints help. Use `--dry-run` first to see what would change before making any modifications.

The current repair routine is JWT key rotation (`--jwt`). It generates a new `HASURA_GRAPHQL_JWT_SECRET`, writes a rotation log entry, and prints the next steps needed to apply the change. The command does not write to `.env.secrets` or restart Hasura automatically. Those steps require explicit operator action to avoid dropping active WebSocket subscriptions unexpectedly.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--jwt` | | bool | `false` | Rotate `HASURA_GRAPHQL_JWT_SECRET` and update the rotation log |
| `--dry-run` | | bool | `false` | Show what would be repaired without making any changes |
| `--to-file` | | string | `""` | Write the new JWT key to this path (mode 0600) instead of stdout |
| `--no-print` | | bool | `false` | Suppress printing the new JWT key to stdout |

## Examples

```bash
# Preview what the JWT rotation would do (no changes)
nself self-heal --dry-run
```

```bash
# Rotate the JWT key and print the new key to stdout
nself self-heal --jwt
```

```bash
# Rotate and write the new key to a file (mode 0600)
nself self-heal --jwt --to-file /tmp/new-jwt.key
```

```bash
# Rotate without printing the key (useful when capturing via --to-file in CI)
nself self-heal --jwt --to-file /run/secrets/jwt.key --no-print
```

```bash
# Dry run combined with file output (preview path without writing)
nself self-heal --jwt --dry-run
```

## See Also

- [cmd-doctor.md](cmd-doctor.md) — run health checks across the full stack
- [cmd-build.md](cmd-build.md) — regenerate docker-compose after applying config changes
- [cmd-restart.md](cmd-restart.md) — restart services after updating secrets
- [Commands.md](Commands.md) — full command index

← [[Commands]] | [[Home]] →
