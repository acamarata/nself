# nself secrets

> Manage encrypted project secrets (age encryption).

## Synopsis

```
nself secrets <subcommand> [--env <env>] [flags] [args]
```

## Description

`nself secrets` manages per-environment encrypted secrets using age encryption. Secrets are stored as age-encrypted JSON files under `.secrets/` (one file per environment). Each team member holds an age keypair; secrets are encrypted to all team members' public keys, so anyone with a valid private key can decrypt without a shared password.

The persistent `--env` flag selects which environment a subcommand operates on (default `dev`). `secrets init` generates the local age keypair and `.secrets/` skeleton. `secrets set / get / list / edit` are the day-to-day operations. `secrets rotate` rolls a value, with `--dual-window` to keep `_PREVIOUS` alongside `_CURRENT` for a transition window; `secrets retire` removes the `_PREVIOUS` half once the rotation has settled.

`secrets schedule` shows the rotation schedule status for tracked secrets. `secrets audit` lists secrets older than the rotation policy. `secrets lint` greps git-tracked files for plaintext secrets. `secrets decrypt-on-deploy` outputs `KEY=VALUE` lines for CI/CD consumption. `secrets rekey --remove <pubkey>` re-encrypts everything without a departed team member's key.

## Subcommands

| Name | Description |
|------|-------------|
| `init` | Generate age key and set up `.secrets/` |
| `set <KEY> [VALUE]` | Set a secret value (prompts if value not provided) |
| `get <KEY>` | Get a secret value |
| `list` | List all secrets for an environment |
| `edit` | Open decrypted secrets in `$EDITOR`, re-encrypt on save |
| `rotate <KEY>` | Rotate a secret to a new value |
| `retire <KEY>` | Retire the `_PREVIOUS` variant of a dual-window rotated secret |
| `schedule` | Show rotation schedule status for all tracked secrets |
| `decrypt-on-deploy` | Output decrypted secrets as `KEY=VALUE` for CI/CD |
| `audit` | Report secrets that have not been rotated in >90 days |
| `lint` | Check for plaintext secrets in git-tracked files |
| `rekey` | Re-encrypt all secrets, removing a team member's key |

## Flags

### Persistent (all subcommands)

| Flag | Default | Description |
|------|---------|-------------|
| `--env` | `dev` | Environment (`dev`, `staging`, `prod`) |

### `secrets rotate <KEY>`

| Flag | Default | Description |
|------|---------|-------------|
| `--dual-window` | false | Keep old key as `_PREVIOUS` during overlap window |

### `secrets rekey`

| Flag | Default | Description |
|------|---------|-------------|
| `--remove` | `""` | Public key to remove from recipients (required) |

## Examples

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

## See Also

- [[cmd-env]], multi-environment management
- [[cmd-backup]], encrypted backups
- [[cmd-license]], license keys
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
