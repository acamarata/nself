# FAQ

## Contents

- [What is nSelf?](#what-is-nself)
- [Is nSelf free?](#is-nself-free)
- [What is the difference between free and Pro plugins?](#what-is-the-difference-between-free-and-pro-plugins)
- [Can I run multiple nSelf projects on one server?](#can-i-run-multiple-nself-projects-on-one-server)
- [How does nSelf compare to Supabase or Nhost?](#how-does-nself-compare-to-supabase-or-nhost)
- [Does nSelf require Docker?](#does-nself-require-docker)
- [What ports does nSelf use?](#what-ports-does-nself-use)
- [How do I update nSelf?](#how-do-i-update-nself)
- [How do I back up my data?](#how-do-i-back-up-my-data)
- [Where are config files stored?](#where-are-config-files-stored)
- [How do I add a custom service?](#how-do-i-add-a-custom-service)
- [Does nSelf support ARM / Apple Silicon?](#does-nself-support-arm--apple-silicon)
- [How do I reset the Hasura admin password?](#how-do-i-reset-the-hasura-admin-password)
- [Can I use nSelf in CI/CD?](#can-i-use-nself-in-cicd)
- [How do I report a security issue?](#how-do-i-report-a-security-issue)
- [How do I contribute a plugin?](#how-do-i-contribute-a-plugin)

## What is nSelf?

nSelf is an open-source CLI that spins up a complete self-hosted backend stack — PostgreSQL, Hasura GraphQL, Auth, and Nginx — in five minutes. You run it on your own server or local machine; there is no cloud dependency.

## Is nSelf free?

Yes. The core CLI and 25 free plugins are MIT licensed and free forever, including commercial use. Revenue comes from 62 paid Pro plugins starting at $9.99/year.

## What is the difference between free and Pro plugins?

Free plugins (25) handle foundational use cases: monitoring, backup, cron, search, notifications, and more. Pro plugins (52) add advanced capabilities: AI, live video (LiveKit), commerce, CMS, SSO/SAML, media transcoding, DRM, WAF, and more. Pro plugins require a license key from [nself.org](https://nself.org).

## Can I run multiple nSelf projects on one server?

Yes. Each project runs in its own Docker network and uses a different `BASE_DOMAIN`. Run `nself init` in separate directories, each with a unique `PROJECT_NAME` and `BASE_DOMAIN`, and start them independently.

## How does nSelf compare to Supabase or Nhost?

nSelf gives you the same Postgres + Hasura + Auth stack as Supabase/Nhost, but self-hosted on your infrastructure with no vendor lock-in. You own your data, choose your server, and pay nothing per row or API call. The tradeoff is that you manage your own server.

## Does nSelf require Docker?

Yes. Docker 24+ with Docker Compose v2 is required. nSelf generates and manages a `docker-compose.yml` for your stack.

## What ports does nSelf use?

Nginx binds to ports 80 and 443. All internal services bind to `127.0.0.1` only and are not directly exposed. Admin UI runs on `localhost:3021`.

## How do I update nSelf?

```bash
nself update
```

This checks for a newer CLI binary and a newer admin Docker image and installs them. Use `nself update --check` to check without installing.

## How do I back up my data?

```bash
nself db backup
```

Creates a `pg_dump` file with a timestamped name. Store it off-server. Restore with `nself db restore /path/to/backup.sql`.

## Where are config files stored?

Project config lives in `.env` in your project directory. The CLI state and license key are stored in `~/.nself/`. Generated files (`docker-compose.yml`, nginx configs, SSL certs) are created inside your project directory.

## How do I add a custom service?

Define `CS_1` through `CS_10` in your `.env`. Choose from 40+ templates or bring your own Docker image.

```bash
CS_1=api:fastapi:3001
```

Then run `nself build && nself restart`.

## Does nSelf support ARM / Apple Silicon?

Yes. Binaries are published for `linux/amd64`, `linux/arm64`, `darwin/amd64`, and `darwin/arm64`. All base Docker images used by nSelf support ARM64.

## How do I reset the Hasura admin password?

The Hasura admin secret is stored as `HASURA_GRAPHQL_ADMIN_SECRET` in your `.env`. Change it there, then run `nself build && nself restart`. Hasura picks up the new value on restart.

## Can I use nSelf in CI/CD?

Yes. Use `--non-interactive` and `--quiet` flags for CI:

```bash
nself init --non-interactive --name myapp
nself build -q
nself start --skip-health-checks --quick
```

## How do I report a security issue?

Email `security@nself.org` or open a private advisory on GitHub. Do not file public issues for security vulnerabilities. See [[Security-Policy]] for full details.

## How do I contribute a plugin?

Read [[Plugin-Dev-Guide]] for the plugin SDK and submission process. Free plugins are accepted via pull request to the `nself-org/plugins` repository. Pro plugins are reviewed privately.

---
← [[Home]] | [[_Sidebar]]
