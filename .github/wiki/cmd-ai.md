# nself ai

> Manage the nSelf AI plugin and local LLM stack.

## Synopsis

```
nself ai <subcommand> [flags] [args]
```

## Description

`nself ai` manages the AI plugin (`plugin-ai`) and the optional local Ollama runtime that powers zero-config inference. It groups three areas: the local Ollama stack (install, models, health, swap, benchmark), a one-shot `chat` for quick verification, and the Gemini API key `pool` for cloud routing.

The `ai local` subtree installs and inspects an Ollama daemon and the small set of models recommended for the host RAM tier. `ai pool` manages auto-provisioned Gemini keys (OAuth-onboarded Google accounts, GCP project creation, key rotation, daily quota tracking) so the AI plugin always has free or near-free capacity.

Most flags are non-destructive. Pulling models requires network access. Pool subcommands talk to the AI plugin over its internal HTTP API; if the plugin is not running, commands report a clear error.

## Subcommands

| Name | Description |
|------|-------------|
| `local install` | Install Ollama, systemd service, firewall, and recommended models |
| `local models list` | List installed and registered local models with diff |
| `local models add <model>` | Pull a model via Ollama and register with plugin-ai |
| `local models remove <model>` | Soft-delete a local model and uninstall from Ollama |
| `local models recommend` | Print recommended models for this host |
| `local health` | Show Ollama and plugin-ai health |
| `local swap <model>` | Hot-swap the default model for a task |
| `local benchmark [model]` | Run benchmark prompts against one or more models |
| `chat <message>` | Send a quick chat message to the local AI |
| `pool init` | Interactive wizard: add a Google account and auto-provision a Gemini key |
| `pool status` | Show pool status (keys, usage, capacity) |
| `pool provision` | Non-interactive provision using stored refresh token |
| `pool add` | Add a Google account via OAuth (opens browser) |
| `pool remove` | Remove a key from the pool (soft-revoke + optional GCP delete) |
| `pool rotate` | Rotate a key (create new GCP key, revoke old) |
| `pool test` | Test one or all keys with a 1-token Gemini request |
| `pool daily-reset` | Manually trigger the daily counter reset |

## Flags

Top-level `nself ai` exposes no flags; flags belong to each subcommand.

### `ai local install`

| Flag | Default | Description |
|------|---------|-------------|
| `--yes` | false | Non-interactive mode |
| `--no-models` | false | Skip model pulls |
| `--model` | `""` | Pull only this model |
| `--bind` | `""` | host:port to bind Ollama to |
| `--json` | false | Emit JSON output |

### `ai local models list`

| Flag | Default | Description |
|------|---------|-------------|
| `--installed` | false | Show only installed |
| `--registered` | false | Show only registered |
| `--json` | false | Emit JSON output |

### `ai local models add <model>`

| Flag | Default | Description |
|------|---------|-------------|
| `--task` | `chat` | Comma-separated task classes |
| `--default` | false | Set as default for the tasks |

### `ai local models remove <model>`

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | false | Remove even if default |

### `ai local models recommend`

| Flag | Default | Description |
|------|---------|-------------|
| `--tier` | `auto` | Force a tier (`auto`, `minimal`, `balanced`, `max`) |

### `ai local health`

| Flag | Default | Description |
|------|---------|-------------|
| `--watch` | false | Re-poll every 2s |
| `--json` | false | Emit JSON output |

### `ai local swap <model>`

| Flag | Default | Description |
|------|---------|-------------|
| `--task` | `chat` | Task: `chat`, `embed`, `classify`, or `all` |
| `--reason` | `""` | Free-text reason (audit log) |

### `ai local benchmark [model]`

| Flag | Default | Description |
|------|---------|-------------|
| `--tasks` | `chat` | Comma-separated tasks |
| `--iterations` | `5` | Iterations per task |

### `ai chat <message>`

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `""` | Model to use (default: `AI_DEFAULT_MODEL` or `gemma2:2b`) |
| `--json` | false | Emit JSON output |

### `ai pool status`

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Emit JSON output |
| `--verbose` | false | Show per-key details |

### `ai pool provision`

| Flag | Default | Description |
|------|---------|-------------|
| `--account` | `""` | Google account email (required) |

### `ai pool add`

| Flag | Default | Description |
|------|---------|-------------|
| `--account` | `""` | Google account email hint |

### `ai pool remove`

| Flag | Default | Description |
|------|---------|-------------|
| `--account` | `""` | Remove by Google account email |
| `--key-id` | `""` | Remove by key index |

### `ai pool rotate`

| Flag | Default | Description |
|------|---------|-------------|
| `--key-id` | `""` | Key index to rotate (required) |

### `ai pool test`

| Flag | Default | Description |
|------|---------|-------------|
| `--key-id` | `""` | Test a specific key |
| `--all` | false | Test all keys |

### `ai pool daily-reset`

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | false | Show what would reset without resetting |

## Examples

```bash
# Install Ollama and pull recommended models for this host
nself ai local install --yes

# Send a quick verification chat
nself ai chat "hello"

# See what models are recommended for the host RAM tier
nself ai local models recommend

# Pull and register a model, set as default for chat
nself ai local models add llama3.2:3b --default

# Onboard a new Google account and auto-provision a Gemini key
nself ai pool init

# Check pool capacity
nself ai pool status --verbose
```

## See Also

- [[cmd-claw]] — nClaw assistant control plane
- [[cmd-doctor]] — diagnose AI runtime issues
- [[cmd-plugin]] — install or update the AI plugin
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
