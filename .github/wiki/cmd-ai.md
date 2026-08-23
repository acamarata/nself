# nself ai

<!-- BEGIN PROSE:summary -->
> Manage the ɳSelf AI plugin and local LLM stack.
<!-- END PROSE:summary -->

## Synopsis

```
nself ai <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself ai` manages the local Ollama runtime that powers zero-config inference. It groups three areas: the local Ollama stack (install, models, health, swap, benchmark), a one-shot `chat` for quick verification, and the Gemini API key `pool` for cloud routing.

For provider key management and AI request routing, use `nself gateway` (wired to `nself-ai-gateway` at port 3761). For PTY session relay commands, use `nself claw session` (wired to `nself-ai-cc` at port 3760).

The `ai local` subtree installs and inspects an Ollama daemon and the small set of models recommended for the host RAM tier. `ai pool` manages auto-provisioned Gemini keys (OAuth-onboarded Google accounts, GCP project creation, key rotation, daily quota tracking) so the gateway always has free or near-free capacity.

Most flags are non-destructive. Pulling models requires network access. Pool subcommands talk to the `nself-ai-gateway` plugin over its internal HTTP API; if the plugin is not running, commands report a clear error.

Top-level `nself ai` exposes no flags; flags belong to each subcommand.
### `ai local install`
### `ai local models list`
### `ai local models add <model>`
### `ai local models remove <model>`
### `ai local models recommend`
### `ai local health`
### `ai local swap <model>`
### `ai local benchmark [model]`
### `ai chat <message>`
### `ai pool status`
### `ai pool provision`
### `ai pool add`
### `ai pool remove`
### `ai pool rotate`
### `ai pool test`
### `ai pool daily-reset`
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `chat` | Send a quick chat message to the local AI |
| `local` | Manage local Ollama runtime and models |
| `pool` | Manage the Gemini API key pool (auto-provisioned + manual) |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
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
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-claw]], ɳClaw assistant control plane
- [[cmd-doctor]], diagnose AI runtime issues
- [[cmd-plugin]], install or update the AI plugin
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
