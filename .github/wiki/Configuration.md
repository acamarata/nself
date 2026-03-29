# Configuration

## Contents

- [Environment Files](#environment-files)
- [Loading Cascade](#loading-cascade)
- [Switching Environments](#switching-environments)
- [Required vs. Auto-Generated Values](#required-vs-auto-generated-values)
- [Validating Configuration](#validating-configuration)
- [Setting Up a New Project](#setting-up-a-new-project)
- [Security Notes](#security-notes)
- [Sub-pages](#sub-pages)

nSelf uses a layered `.env` file system for all configuration. Every service in your stack — Postgres, Hasura, Auth, Nginx, and any optional services — is driven entirely by environment variables. There is no YAML-based config format to learn; you set variables in `.env` files and the CLI generates the correct `docker-compose` output automatically.

This page explains which files exist, what each one is for, how they stack together, and how to work with them day to day.

---

## Environment Files

| File | Purpose | Committed to git |
|------|---------|-----------------|
| `.env.dev` | Team defaults — safe base values for local development | Yes |
| `.env.staging` | Staging overrides applied when `ENV=staging` | Yes |
| `.env.prod` | Production overrides (non-secret values only) | Yes |
| `.env.secrets` | Production secrets — passwords, API keys, tokens | **No (gitignored)** |
| `.env.local` | Personal local overrides — your machine only | **No (gitignored)** |
| `.env` | Highest-priority override — takes precedence over everything | **No (gitignored)** |

The three committed files (`.env.dev`, `.env.staging`, `.env.prod`) contain non-sensitive configuration that the whole team can share. Secret material — database passwords, admin secrets, JWT keys — belongs in `.env.secrets` or `.env.local`, which are always gitignored.

---

## Loading Cascade

When nSelf starts, it loads environment files in a fixed order. **Each file that is loaded overrides any variables already set by earlier files.** Files that do not exist are silently skipped.

```
1.  .env.dev        ← base team defaults (always loaded)
2.  .env.staging    ← loaded when ENV=staging
3.  .env.prod       ← loaded when ENV=prod
4.  .env.secrets    ← production secrets (gitignored)
5.  .env.local      ← personal local overrides (gitignored)
6.  .env            ← highest-priority override (gitignored)
```

After all files have been loaded, `apply_smart_defaults()` runs and fills in any variables that were not set by any file. This means you only need to specify the values you actually want to change — everything else gets a sensible default automatically.

A practical example: `.env.dev` sets `HASURA_GRAPHQL_ENABLE_CONSOLE=true`. Your `.env.prod` sets `HASURA_GRAPHQL_ENABLE_CONSOLE=false`. When you run with `ENV=prod`, the production file is loaded after the dev file, so the console is disabled. Your `.env.local` could override that again for a specific machine if needed.

---

## Switching Environments

Pass the `-e` flag (or `--env`) to any command that spins up services:

```bash
# Local development (default — same as omitting -e)
nself start -e dev

# Staging environment
nself start -e staging

# Production environment
nself start -e prod
```

The `ENV` variable also accepts common aliases so you do not need to remember the canonical names:

| You type | Resolved to |
|----------|-------------|
| `development`, `develop`, `devel` | `dev` |
| `production`, `prod` | `prod` |
| `staging`, `stage` | `staging` |

You can also set `ENV` directly in your `.env` or `.env.local` file if you always work in the same environment on a given machine:

```bash
# .env.local
ENV=dev
```

---

## Required vs. Auto-Generated Values

Some variables are **required** — nSelf will refuse to start if they are missing:

- `PROJECT_NAME` — your project's namespace (lowercase, 2–30 characters)
- `POSTGRES_PASSWORD` — must be at least 16 characters
- `HASURA_GRAPHQL_ADMIN_SECRET` — must be at least 32 characters
- `HASURA_JWT_KEY` — must be at least 32 characters

Other values are **auto-generated** the first time you run `nself init`. The CLI writes them into your `.env` file so they persist across restarts. You should not regenerate these after a project is live — doing so would invalidate all existing sessions and tokens.

When in doubt, run the validation command (see below) to check which required variables are missing.

---

## Validating Configuration

Before starting your stack — especially before deploying to production — validate your configuration:

```bash
nself config validate
```

This command loads the full cascade for your current environment and checks every required variable. It reports:

- Missing required values
- Values that fail format validation (e.g., a password that is too short)
- Variables that reference other undefined variables
- Security warnings for production environments (such as wildcard CORS origins)

To validate a specific environment without starting any services:

```bash
nself config validate -e prod
```

You can also print the resolved configuration (all variables after the cascade) to inspect what values the CLI will actually use:

```bash
nself config show
nself config show -e prod
```

---

## Setting Up a New Project

For a brand-new project, the typical workflow is:

```bash
# 1. Initialise — creates .env.dev with defaults and auto-generates secrets
nself init

# 2. Edit .env.dev to set your project name and domain
#    PROJECT_NAME=myproject
#    BASE_DOMAIN=myproject.local

# 3. For production, copy the secrets template and fill it in
cp .env.secrets.example .env.secrets
# Edit .env.secrets — set your real passwords and keys

# 4. Validate before starting
nself config validate

# 5. Start the stack
nself start
```

---

## Security Notes

- **Never commit `.env.secrets`** — it is gitignored by default, but double-check before pushing
- **Never put secrets in `.env.dev`** — this file is committed and shared with your team
- In production, set `HASURA_GRAPHQL_ENABLE_CONSOLE=false` and `HASURA_GRAPHQL_DEV_MODE=false`
- Use strong, randomly generated passwords (at least 32 characters) for `POSTGRES_PASSWORD`, `HASURA_GRAPHQL_ADMIN_SECRET`, and `HASURA_JWT_KEY`
- Do not reuse the same secrets across projects or environments

---

## Sub-pages

- [[Config-Env-Vars]] — Complete environment variable reference for every service
- [[Config-Postgres]] — PostgreSQL configuration in depth
- [[Config-Hasura]] — Hasura GraphQL Engine configuration
- [[Config-Auth]] — Authentication service configuration
- [[Config-Nginx]] — Nginx reverse proxy and SSL configuration
- [[Config-Optional-Services]] — Redis, MinIO, Functions, Search, Mail, MLflow, Admin
- [[Config-Custom-Services]] — Custom service slots (CS_1 through CS_10)

---

← [[Home]] | [[Getting-Started]] | [[Config-Env-Vars]] →
