# GitHub Plugin

> GitHub integration — repos, issues, PRs, workflow events, and OAuth. **Free — MIT licensed.**

## Install

```bash
nself plugin install github
```

## What It Does

Syncs GitHub data into your Postgres database: repositories, issues, pull requests, commits, workflow runs, and webhook events. Provides a webhook receiver for real-time GitHub events. Supports OAuth authentication via GitHub for user login. Exposes 23 database tables with full GitHub data models.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `GITHUB_PORT` | `3002` | GitHub plugin port |
| `GITHUB_TOKEN` | — | GitHub personal access token or App token |
| `GITHUB_WEBHOOK_SECRET` | — | Webhook HMAC secret |
| `GITHUB_APP_ID` | — | GitHub App ID (if using App auth) |
| `GITHUB_APP_PRIVATE_KEY` | — | GitHub App private key |

## Ports

| Port | Purpose |
|------|---------|
| 3002 | GitHub plugin REST API and webhook receiver |

## Database Tables

23 tables added to your Postgres database, including:
- `np_github_repos` — repository metadata
- `np_github_issues` — issues and comments
- `np_github_pull_requests` — PRs and reviews
- `np_github_commits` — commit history
- `np_github_workflow_runs` — CI/CD run history
- `np_github_webhook_events` — raw webhook event log
- And 17 more for stars, labels, milestones, etc.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/github/webhook` | GitHub webhook receiver |
| `/github/oauth/callback` | OAuth callback |

## API

```
GET  /health           — Health check
POST /sync/repo/{owner}/{repo} — Sync a repository
GET  /repos            — List synced repositories
POST /webhook          — GitHub webhook receiver
GET  /oauth/authorize  — Start GitHub OAuth flow
```
