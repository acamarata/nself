# nself init

> Initialize a new nSelf project with an interactive configuration wizard.

## Synopsis

```
nself init [flags]
```

## Description

`nself init` launches an interactive setup wizard that creates a pristine `.env` configuration for a new nSelf project. It prompts for your project name, base domain, and email, then auto-generates cryptographically secure secrets (Postgres password, Hasura admin secret, JWT key).

You can choose which optional services to enable during init (Redis, MinIO, MeiliSearch, Mailpit, Monitoring). Each selection updates the generated `.env` accordingly. All values are validated before being written — domain format, password strength, and required fields are all checked.

After `nself init` completes, run `nself build` to generate `docker-compose.yml` and nginx configs, then `nself start` to boot the stack.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--wizard` | false | Run the full 10-step interactive wizard |
| `--interactive` | false | Explicitly enable interactive wizard |
| `--non-interactive` | false | Use all defaults without prompts (CI-safe) |
| `--fast` | false | Skip advanced options, use smart defaults |
| `--demo` | false | Auto-configure with all services enabled |
| `--full` | false | Create `.env.dev`, `.env.staging`, `.env.prod`, `.env.secrets` |
| `--force` | false | Overwrite existing configuration |
| `--template` | `""` | Use a specific template: `express`, `fastapi`, `go`, `rust` |
| `--name` | `""` | Project name (sets `PROJECT_NAME` in generated `.env`) |
| `--domain` | `""` | Base domain (skips interactive domain prompt, e.g. `myapp.dev`) |
| `--skip-validation` | false | Skip configuration validation |
| `--quiet` | false | Suppress output messages |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Minimal interactive setup
nself init

# Full 10-step wizard
nself init --wizard

# All services enabled (demo/evaluation)
nself init --demo

# Create all env files at once
nself init --full

# Smart defaults, no prompts
nself init --fast

# Non-interactive — all defaults, safe for CI
nself init --non-interactive

# Skip prompts by supplying name and domain directly
nself init --name myapp --domain myapp.dev

# Start from a Go project template
nself init --template go --name myapi --domain myapi.local
```

← [[Commands]] | [[Home]] →
