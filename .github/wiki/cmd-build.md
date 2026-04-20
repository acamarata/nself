# nself build

> Generate `docker-compose.yml`, nginx configs, and SSL certificates from `.env`.

## Synopsis

```
nself build [flags]
```

## Description

`nself build` reads your `.env` cascade and generates all infrastructure configuration files: a `docker-compose.yml` with every enabled service, nginx reverse-proxy configs, and SSL certificates. It must be run after `nself init` and after any configuration change before restarting services.

The build pipeline loads configuration from `.env.dev` → `.env.{ENV}` → `.env.secrets` → `.env.local` → `.env`, merges plugin configurations from `~/.nself/plugins/`, and applies security validation (password strength, no wildcard CORS in production, port binding checks). The result is a single `docker-compose.yml` that includes core services, optional services, monitoring, custom services (CS_1–CS_10), and any installed plugins.

By default, `nself build` is smart-cached: it compares `.env` modification time against `docker-compose.yml` and skips regeneration when nothing has changed. Use `--force` to override the cache.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--force`, `-f` | false | Force rebuild all components, ignore cache |
| `--check` | false | Validate configuration only — do not write any files |
| `--verbose`, `-v` | false | Show environment cascade during build |
| `--quiet`, `-q` | false | Suppress non-error output (CI use) |
| `--no-cache` | false | Disable build cache |
| `--debug` | false | Enable debug mode |
| `--allow-insecure` | false | Allow insecure configuration (dev only) |
| `--security-report` | false | Generate a security analysis after build |
| `--no-migration-check` | false | Skip v0.9 artifact detection (automation/CI) |
| `--allow-legacy` | false | Bypass v0.9 artifact check and proceed with WARNING (not recommended). Use only as a temporary workaround while running `nself migrate`. |
| `--no-monorepo` | false | Disable automatic monorepo backend detection |
| `--help`, `-h` | — | Show help |

## v0.9 project detection

`nself build` scans for v0.9 project artifacts before generating any files. Two or more detected artifacts trigger a hard error pointing to the migration guide. A single artifact produces a non-blocking warning. Use `--no-migration-check` in automation (CI) where you are certain no v0.9 projects exist. Use `--allow-legacy` only as a temporary workaround while running `nself migrate`. See [[Upgrade-From-v0.9]].

## Examples

```bash
# Standard build
nself build

# Validate config only, don't write files
nself build --check

# Force rebuild everything, ignoring cache
nself build --force

# CI mode — quiet output
nself build -q

# Show the environment cascade as it loads
nself build --verbose

# Generate a security analysis report
nself build --security-report

# Rebuild for a specific environment
nself build --force --verbose
```

← [[Commands]] | [[Home]] →
