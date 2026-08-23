# nself model

<!-- BEGIN PROSE:summary -->
> Manage local AI models via Ollama.
<!-- END PROSE:summary -->

## Synopsis

```
nself model <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
**Manage local AI models via Ollama.**

The `model` command provides a focused surface for day-to-day local model operations: list
what is installed, pull new models, remove ones you no longer need, update to the latest
tag, and benchmark inference speed.

All subcommands talk to the Ollama API. Ollama must be running:

nself plugin install ollama
nself start

---

---
## nself model list

```
nself model list [--json]
```

Show every model currently downloaded in the local Ollama store.

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Emit JSON array instead of formatted table |

### Output columns

| Column | Description |
|--------|-------------|
| NAME | Ollama tag (e.g. `llama3.2:3b`) |
| SIZE | Compressed disk size |
| MODIFIED | Date last pulled or updated |
| (tag) | `[default]` when the model matches `NSELF_OLLAMA_DEFAULT_MODEL` |

### Examples

```bash
nself model list

nself model list --json | jq '.[].name'
```

```
NAME                                        SIZE  MODIFIED
------------------------------------------------------------------------
gemma-3-4b                               2.5 GB  2026-04-01    [default]
llama3.2:3b                              1.9 GB  2026-04-10
```

---

## nself model pull

```
nself model pull <model>
```

Download a model from the Ollama library. Model names follow the Ollama tag format:

| Format | Example |
|--------|---------|
| `<name>` | `llama3.2` (latest tag) |
| `<name>:<tag>` | `llama3.2:3b` (specific size/quant) |

Common models and approximate hardware requirements:

| Model | Size | Min RAM | Notes |
|-------|------|---------|-------|
| `gemma-3-4b` | ~2.5 GB | 4 GB | Good for CPU-only inference |
| `llama3.2:3b` | ~2.0 GB | 4 GB | Fast general chat |
| `llama3.2:7b` | ~4.7 GB | 8 GB | Higher quality |
| `mistral` | ~4.1 GB | 8 GB | Strong instruct model |

### Examples

```bash
nself model pull gemma-3-4b
nself model pull llama3.2:3b
nself model pull mistral
```

```
Pulling llama3.2:3b...
Model llama3.2:3b ready.
```

---

## nself model remove

```
nself model remove <model>
nself model rm <model>
nself model delete <model>
```

Delete a downloaded model from the local Ollama store. The disk space is freed immediately.

### Examples

```bash
nself model remove gemma-3-4b
nself model rm llama3.2:7b
```

```
Removed gemma-3-4b.
```

---

## nself model update

```
nself model update <model>
```

Re-pull a model to pick up the newest weights for its tag. Ollama only downloads layers
that have changed, so updates are incremental.

### Examples

```bash
nself model update gemma-3-4b
```

```
Updating gemma-3-4b...
Pulling gemma-3-4b...
Model gemma-3-4b ready.
```

---

## nself model benchmark

```
nself model benchmark <model> [--runs N] [--prompt "..."] [--json]
```

Send a standard prompt to the model N times and report:

| Metric | Description |
|--------|-------------|
| `avg latency` | Mean response time across all successful runs |
| `p99 latency` | 99th-percentile response time |
| `tok/s` | Average tokens per second (total tokens / total elapsed) |
| `total tokens` | Sum of response tokens across all runs |
| `errors` | Count of failed inference calls |

The default prompt is: `Explain what a Merkle tree is in two sentences.`

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--runs` | `5` | Number of inference runs |
| `--prompt` | (Merkle tree question) | Custom prompt string |
| `--json` | off | Emit JSON output |

### Examples

```bash
# Basic benchmark
nself model benchmark gemma-3-4b

# 20-run benchmark for a stable p99
nself model benchmark llama3.2:3b --runs 20

# Custom prompt
nself model benchmark mistral --prompt "Summarize TCP in one sentence."

# JSON output for scripting
nself model benchmark gemma-3-4b --json | jq '.avg_tok_s'
```

```
Benchmarking gemma-3-4b  (5 runs)...

  Model:        gemma-3-4b
  Runs:         5 / 5 succeeded
  Avg latency:  1842 ms
  p99 latency:  2103 ms
  Tok/s:        14.2
  Total tokens: 143
  Elapsed:      9.21s
```

### JSON schema

```json
{
  "model":        "gemma-3-4b",
  "runs":         5,
  "avg_tok_s":    14.2,
  "p99_ms":       2103.0,
  "total_tokens": 143,
  "errors":       0
}
```

---

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NSELF_OLLAMA_HOST` | `http://localhost:11434` | Ollama API base URL |
| `PLUGIN_AI_OLLAMA_URL` | `http://localhost:11434` | Fallback if `NSELF_OLLAMA_HOST` is unset |
| `NSELF_OLLAMA_TIMEOUT_SECONDS` | `120` | Per-request timeout in seconds |
| `NSELF_OLLAMA_DEFAULT_MODEL` | (unset) | Model name highlighted as default in `list` |

---
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
| `benchmark` | Run a standard prompt and report tokens/s + p99 latency |
| `list` | List all pulled Ollama models |
| `ollama` | Manage local Ollama LLM stack |
| `pull` | Pull (download) a model from the Ollama registry |
| `remove` | Remove a pulled model from Ollama to free disk space |
| `update` | Re-pull a model to fetch the latest version of its tag |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
nself model list
  nself model pull llama3.2:3b
  nself model remove gemma-3-4b
  nself model update llama3.2:3b
  nself model benchmark llama3.2:3b
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-ollama]], manage the Ollama service container (install, status)
- [[cmd-ai]], manage models via the `nself-ai-gateway` registry
- [[cmd-plugin]], install and manage ɳSelf plugins
- [[Home]]
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
