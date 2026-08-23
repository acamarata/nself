# nself config

<!-- BEGIN PROSE:summary -->
> View and manage project configuration.
<!-- END PROSE:summary -->

## Synopsis

```
nself config <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself config` provides a complete interface for reading and writing ɳSelf configuration values. Config is stored in `.env` files, `nself config` reads and writes them safely, preserving comments and formatting.

Secret keys (any key containing `SECRET`, `PASSWORD`, `KEY`, or `TOKEN`) are masked as `***` in output by default. Pass `--reveal` to show plaintext values. Use `--env` to target a specific environment file (e.g., `--env staging` reads `.env.staging`).

The `validate` subcommand runs all registered configuration validators and reports pass/fail per rule. It is automatically run during `nself build`, but you can run it independently to check config without rebuilding.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--env` | `""` | Target environment (reads .env.{env}) |
| `--json` | `false` | JSON output |
| `--reveal` | `false` | Show secret values in plaintext |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `export` | Export current config to a file or stdout |
| `features` | Manage CLI-built-in feature flags |
| `get` | Get a single configuration value |
| `import` | Import config from a file into .env |
| `list` | List all known config keys with current values |
| `set` | Update a configuration value (writes to .env) |
| `show` | Show all config key=value pairs (masked by default) |
| `validate` | Validate configuration against all registered rules |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Show all config (secrets masked)
nself config show

# Show config with secrets revealed
nself config show --reveal

# Show config for staging environment
nself config show --env staging

# Get a single value
nself config get BASE_DOMAIN

# Get a secret value
nself config get POSTGRES_PASSWORD --reveal

# Set a value
nself config set BASE_DOMAIN myapp.dev
nself config set REDIS_ENABLED true

# List all known keys with current values and source
nself config list
nself config list --env staging

# Validate configuration
nself config validate
nself config validate --env prod

# Export config to a file
nself config export backup.env
nself config export /tmp/prod-config.env --env prod

# Import config from a file
nself config import backup.env
nself config import staging.env --env staging --force
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
