# nself secrets

<!-- BEGIN PROSE:summary -->
> Manage encrypted project secrets (age encryption).
<!-- END PROSE:summary -->

## Synopsis

```
nself secrets <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself secrets` manages per-environment encrypted secrets using age encryption. Secrets are stored as age-encrypted JSON files under `.secrets/` (one file per environment). Each team member holds an age keypair; secrets are encrypted to all team members' public keys, so anyone with a valid private key can decrypt without a shared password.

The persistent `--env` flag selects which environment a subcommand operates on (default `dev`). `secrets init` generates the local age keypair and `.secrets/` skeleton. `secrets set / get / list / edit` are the day-to-day operations. `secrets rotate` rolls a value, with `--dual-window` to keep `_PREVIOUS` alongside `_CURRENT` for a transition window; `secrets retire` removes the `_PREVIOUS` half once the rotation has settled.

`secrets schedule` shows the rotation schedule status for tracked secrets. `secrets audit` lists secrets older than the rotation policy. `secrets lint` greps git-tracked files for plaintext secrets. `secrets decrypt-on-deploy` outputs `KEY=VALUE` lines for CI/CD consumption. `secrets rekey --remove <pubkey>` re-encrypts everything without a departed team member's key.

### Persistent (all subcommands)
### `secrets rotate <KEY>`
### `secrets rekey`
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--env` | `dev` | Environment (dev, staging, prod) |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `audit` | Report secrets that haven't been rotated in >90 days |
| `decrypt-on-deploy` | Output decrypted secrets as KEY=VALUE for CI/CD |
| `edit` | Open decrypted secrets in $EDITOR, re-encrypt on save |
| `get` | Get a secret value |
| `init` | Generate age key and set up .secrets/ directory |
| `lint` | Check for plaintext secrets in git-tracked files |
| `list` | List all secrets for an environment |
| `list-schedules` | List all rotation schedule statuses (alias for: secrets schedule) |
| `rekey` | Re-encrypt all secrets, removing a team member's key |
| `retire` | Retire the _PREVIOUS variant of a dual-window rotated secret |
| `rotate` | Rotate a secret to a new value |
| `rotation-log` | Show rotation event log |
| `schedule` | Add a rotation schedule or show all schedule statuses |
| `set` | Set a secret value (prompts if value not provided) |
| `verify` | Verify that a named secret exists in the store |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Bootstrap encryption for a new project member
nself secrets init

# Set a secret in the dev environment
nself secrets set STRIPE_SECRET_KEY sk_test_xxx

# Set a secret in production by prompt
nself secrets set --env prod STRIPE_SECRET_KEY

# Show all secret keys (values redacted)
nself secrets list --env prod

# Rotate a key with a 7-day dual-window
nself secrets rotate --env prod JWT_SIGNING_KEY --dual-window

# After verifying the new key works, retire the old one
nself secrets retire --env prod JWT_SIGNING_KEY

# Audit which secrets are overdue for rotation
nself secrets audit --env prod

# Check git-tracked files for plaintext leakage
nself secrets lint

# Remove a departed team member's pubkey and re-encrypt everything
nself secrets rekey --remove "age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-env]], multi-environment management
- [[cmd-backup]], encrypted backups
- [[cmd-license]], license keys
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
