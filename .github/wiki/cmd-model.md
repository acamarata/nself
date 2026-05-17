# nself model

> List, pull, remove, update, and benchmark local AI models via Ollama.

## Synopsis

```
nself model <subcommand> [flags]
```

## Description

`nself model` manages AI models stored in the local Ollama model registry. It covers the full model lifecycle: browsing what is installed, downloading new models, removing models to reclaim disk space, re-pulling for the latest weights, and running a repeatable latency and throughput benchmark.

All subcommands talk to the Ollama API. The host is resolved from `NSELF_OLLAMA_HOST`, falling back to `PLUGIN_AI_OLLAMA_URL`, and defaulting to `http://localhost:11434`. The Ollama plugin must be installed and the stack running:

```bash
nself plugin install ollama
nself start
```

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `list` | Show all pulled models with name, size, and modification date |
| `pull` | Download a model from the Ollama registry |
| `remove` | Delete a model from the local store to free disk space |
| `update` | Re-pull a model to pick up the latest tag version |
| `benchmark` | Run a standard prompt N times and report tok/s and p99 latency |

## Flags

### `nself model list`

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Emit JSON output instead of the table |

### `nself model pull`

No flags beyond the required positional argument.

### `nself model remove`

Aliases: `rm`, `delete`. No flags beyond the required positional argument.

### `nself model update`

No flags beyond the required positional argument.

### `nself model benchmark`

| Flag | Default | Description |
|------|---------|-------------|
| `--prompt` | `""` | Custom prompt to use (default: Merkle tree question) |
| `--runs` | `5` | Number of inference runs (higher = more stable p99) |
| `--json` | false | Emit JSON output |

## Environment variables

| Variable | Description |
|----------|-------------|
| `NSELF_OLLAMA_HOST` | Full base URL for the Ollama API (e.g. `http://localhost:11434`) |
| `PLUGIN_AI_OLLAMA_URL` | Alternative URL shared with the `nself ai` command tree |
| `NSELF_OLLAMA_DEFAULT_MODEL` | Model name marked as `[default]` in `nself model list` |
| `NSELF_OLLAMA_TIMEOUT_SECONDS` | Request timeout in seconds (default: 120) |

## Examples

```bash
# List all downloaded models
nself model list

# List as JSON for scripting
nself model list --json

# Pull the gemma-3-4b model (good for CPU-only machines)
nself model pull gemma-3-4b

# Pull a specific tag
nself model pull llama3.2:3b

# Remove a model to free disk
nself model remove gemma-3-4b
nself model rm llama3.2:3b

# Re-pull to pick up updated weights
nself model update llama3.2:3b

# Benchmark with the default prompt (5 runs)
nself model benchmark llama3.2:3b

# Benchmark with a custom prompt and more runs
nself model benchmark llama3.2:3b --prompt "Explain backpressure in 3 sentences." --runs 20

# Benchmark output as JSON
nself model benchmark llama3.2:3b --json
```

## Common models

| Model | Size | Notes |
|-------|------|-------|
| `gemma-3-4b` | ~2.5 GB | Good for CPU-only inference |
| `llama3.2:3b` | ~2.0 GB | Fast general chat |
| `llama3.2:7b` | ~4.7 GB | Higher quality, needs 8 GB RAM |
| `mistral` | ~4.1 GB | Good instruct model |

## See Also

- [[cmd-ai]] — AI assistant commands (uses the default model)
- [[cmd-claw]] — ɳClaw plugin management
- [[Plugin-Overview]] — list of AI plugins

← [[Commands]] | [[Home]] →
