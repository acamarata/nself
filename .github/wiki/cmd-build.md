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
| `--check` | false | Validate configuration only , do not write any files |
| `--verbose`, `-v` | false | Show environment cascade during build |
| `--quiet`, `-q` | false | Suppress non-error output (CI use) |
| `--no-cache` | false | Disable build cache |
| `--debug` | false | Enable debug mode |
| `--allow-insecure` | false | Allow insecure configuration (dev only) |
| `--security-report` | false | Generate a security analysis after build |
| `--no-migration-check` | false | Skip v0.9 artifact detection (automation/CI) |
| `--allow-legacy` | false | Bypass v0.9 artifact check and proceed with WARNING (not recommended). Use only as a temporary workaround while running `nself migrate`. |
| `--no-monorepo` | false | Disable automatic monorepo backend detection |
| `--no-auto-redis` | false | Disable automatic Redis inclusion when a BullMQ-backed plugin is detected |
| `--help`, `-h` | — | Show help |

## Redis auto-enable

`nself build` automatically includes a Redis service in `docker-compose.yml` when any installed plugin requires it, even if `REDIS_ENABLED` is not set in `.env`.

Plugins that trigger Redis auto-enable: `ai`, `claw`, `mux`, `cron`, `notify`.

When auto-enable fires, the build prints:

```
Note: Redis auto-enabled because a BullMQ-backed plugin (cron, notify, ai, claw, or mux) was detected.
```

To disable auto-enable, uninstall the plugin or explicitly set `REDIS_ENABLED=false` and confirm you do not need the plugin's queue features. To opt in explicitly and suppress the note, set `REDIS_ENABLED=true` in `.env`.

## v0.9 project detection

`nself build` scans for v0.9 project artifacts before generating any files. Two or more detected artifacts trigger a hard error pointing to the migration guide. A single artifact produces a non-blocking warning. Use `--no-migration-check` in automation (CI) where you are certain no v0.9 projects exist. Use `--allow-legacy` only as a temporary workaround while running `nself migrate`. See [[Upgrade-From-v0.9]].

## Docker Compose v5 compatibility

`nself build` generates compose files that are compatible with both Docker Compose v4 and v5.

Docker Compose v5 introduced a stricter parser: it treats the top-level `pids_limit` service field as a canonical alias for `deploy.resources.limits.pids`. When both are present, v5 rejects the compose file with:

```
services.<name>: can't set distinct values on 'pids_limit' and 'deploy.resources.limits.pids': invalid compose project
```

The generator uses the `deploy.resources.limits.pids` form exclusively. The `pids_limit` top-level field is never emitted. This form is also valid on Compose v3.4+ and v4, so generated files work across all current Docker Compose versions.

Each long-running service gets a default pids limit of 100 to prevent fork-bomb attacks. Services that need more (such as postgres under high concurrency) override this in their configuration.

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
