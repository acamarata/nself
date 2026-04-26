# GitHub Runner Plugin

> Self-hosted GitHub Actions runner, register your ɳSelf host as a CI runner. **Free, MIT licensed.**

## Install

```bash
nself plugin install github-runner
```

## What It Does

Registers your ɳSelf server as a self-hosted GitHub Actions runner. Runs CI/CD workflows on your own infrastructure instead of GitHub-hosted runners. Supports runner groups, custom labels, and multiple runner instances.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `GITHUB_RUNNER_TOKEN` | — | GitHub runner registration token |
| `GITHUB_RUNNER_REPO` | — | Repository URL (e.g., `https://github.com/org/repo`) |
| `GITHUB_RUNNER_NAME` | `nself-runner` | Runner display name |
| `GITHUB_RUNNER_LABELS` | `self-hosted,linux` | Comma-separated runner labels |
| `GITHUB_RUNNER_COUNT` | `1` | Number of runner instances |

## Ports

None, this plugin runs as a background process, not an HTTP server.

## Database Tables

0 tables, runner state is managed by GitHub and local files.

## Nginx Routes

None.

## Usage

After install and configuration:

```bash
nself build && nself start
```

The runner registers automatically on start. Verify in your GitHub repository under **Settings → Actions → Runners**.
