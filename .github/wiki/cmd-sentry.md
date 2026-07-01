# nself sentry

> Operate ɳSentry monitoring from the CLI: monitors, incidents, status pages, and alert channels against the hosted SaaS or a self-hosted bundle.

## Synopsis

```
nself sentry <subcommand> [flags]
```

## Description

`nself sentry` is the CLI surface for ɳSentry. The `status` subcommand reads a public
status page. Every other subcommand talks to the ɳSentry REST API, either the hosted
SaaS at `https://api.sentry.nself.org` (default) or a self-hosted/local sentry bundle.

| Subcommand | Purpose |
|------------|---------|
| `login` | Validate and store an API key at `~/.nself/sentry.json` (mode 0600) |
| `logout` | Remove the stored API key |
| `whoami` | Show account, tier, and quota usage |
| `monitors list\|add\|rm\|pause\|resume` | Manage uptime monitors |
| `incidents list\|ack\|resolve` | Incident lifecycle |
| `status-pages list\|create` | Manage public status pages |
| `alerts channels\|test` | List alert channels, send test notifications |
| `status` | Fetch a status page JSON and report component health |

### Authentication

API keys start with `nsk_` and are created in the ɳSentry dashboard, or seeded locally
by the sentry dev preset. Resolution order for both the key and the API URL:

1. `--api-key` / `--api-url` flags
2. `NSELF_SENTRY_API_KEY` / `NSELF_SENTRY_API_URL` environment variables
3. `~/.nself/sentry.json` (written by `nself sentry login`)
4. Default API URL `https://api.sentry.nself.org`

A 401 response always suggests `nself sentry login`. Quota errors (402/429) include the
server message and an upgrade hint.

### The nsentry alias

A symlink named `nsentry` behaves as this command group:

```bash
ln -s $(which nself) /usr/local/bin/nsentry
nsentry monitors list    # same as: nself sentry monitors list
```

### JSON output

Every cloud subcommand accepts `--json` for script and AI-agent consumption. The same
operations are also exposed as MCP tools by [[cmd-mcp]] (`sentry_monitors_list`,
`sentry_monitors_add`, `sentry_incidents_list`, `sentry_incidents_ack`, `sentry_status`).

## Flags

Common to all cloud subcommands:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--api-url` | string | `https://api.sentry.nself.org` | ɳSentry API base URL |
| `--api-key` | string | stored key | ɳSentry API key (`nsk_*`) |
| `--json` | bool | `false` | Output JSON instead of tables |

`monitors add` flags:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--url` | string | (required) | Target URL or host to monitor |
| `--name` | string | the URL | Display name |
| `--kind` | string | `http` | Monitor kind: `http`, `tcp`, or `ping` |
| `--interval` | duration | `60s` | Check interval (tier floors: free 5m, bundle 1m, ɳSelf+ 30s) |

## Examples

```bash
# Log in to the hosted SaaS
nself sentry login --api-key nsk_abc123...
```

```bash
# Add a monitor and list monitors
nself sentry monitors add --name api --url https://api.example.com --interval 60s
nself sentry monitors list
```

```bash
# Triage incidents
nself sentry incidents list --status open
nself sentry incidents ack inc_123
nself sentry incidents resolve inc_123
```

```bash
# Status pages and alert channels
nself sentry status-pages create --name "Public Status" --slug public
nself sentry alerts channels
nself sentry alerts test ch_email_1
```

```bash
# Account, tier, and quota usage as JSON
nself sentry whoami --json
```

```bash
# Fully local, zero cloud: sentry dev preset seeds a tenant + dev API key
nself init --preset sentry && nself build && nself start
nself sentry login --api-url http://localhost:3848 --api-key nsk_dev_local_0000000000000000
nself sentry monitors add --url https://example.com
```

## Tier Quotas

| | Free | Sentry Bundle | ɳSelf+ |
|---|---|---|---|
| Monitors | 10 at 5-min | 50 at 1-min | 100 at 30s |
| Status pages | 1 | 3 | 10 |
| API/CLI/MCP | read-only | full | full |

## See Also

- [[cmd-mcp]] — MCP server with the `sentry_*` tools for AI agents
- [[cmd-init]] — `--preset sentry` for the local-first dev stack
- [[cmd-doctor]] — nSelf diagnostics

← [[Commands]] | [[Home]] →
