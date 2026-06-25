# nself ai local models

> List, pull, and remove local Ollama models with hardware compatibility markers.

## Synopsis

```
nself ai local models <subcommand> [flags]
```

## Description

`nself ai local models` manages the set of Ollama models on the host. It cross-references the bundled model registry against detected hardware (RAM, VRAM, GPU) and marks each model as supported, marginal, or insufficient for the current machine.

The registry is bundled with the AI plugin binary and never fetched at runtime. Hardware detection works on macOS, Linux, and Windows; Apple Silicon unified memory is handled as a special case (1.5× headroom threshold).

## Subcommands

### `nself ai local models list`

Lists all models from the registry alongside their hardware compatibility markers.

```
nself ai local models list [flags]
```

**Output columns**

| Column | Description |
|--------|-------------|
| Model | Model identifier (e.g. `llama3.1:8b`) |
| Provider | `ollama` / `anthropic` / `google` / `openai` |
| Context | Context window in tokens |
| RAM | Minimum RAM required (GB) |
| Compat | ✅ supported · 🟡 marginal · 🔒 insufficient · ☁ cloud |
| Installed | Whether the model is installed in Ollama |

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--installed` | false | Show only models already installed in Ollama |
| `--json` | false | Output as JSON array |

**Example**

```bash
$ nself ai local models list
MODEL                  PROVIDER   CONTEXT  RAM    COMPAT  INSTALLED
llama3.1:8b            ollama     128k     8 GB   ✅      yes
llama3.1:70b           ollama     128k     40 GB  🔒      no
mistral-nemo           ollama     128k     12 GB  🟡      no
qwen2.5-coder:7b       ollama     128k     6 GB   ✅      yes
claude-opus-4-7        anthropic  200k     —      ☁       —
claude-sonnet-4-6      anthropic  200k     —      ☁       —
gemini-2.5-pro         google     1M       —      ☁       —
```

---

### `nself ai local models add <model>`

Pulls a model from the Ollama library (streaming progress) and registers it with the AI plugin.

```
nself ai local models add <model> [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Output progress events as JSON |

**Example**

```bash
$ nself ai local models add mistral-nemo
Pulling mistral-nemo...
████████████████████░░░ 84%  3.1 GB / 3.7 GB
```

---

### `nself ai local models remove <model>`

Removes a model from Ollama (with confirmation) and unregisters it from the AI plugin.

```
nself ai local models remove <model> [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--yes` | false | Skip confirmation prompt |
| `--json` | false | Output as JSON |

**Example**

```bash
$ nself ai local models remove mistral-nemo
Remove mistral-nemo from Ollama? [y/N] y
Removed.
```

---

### `nself ai local models recommend`

Prints the models that the registry recommends for the detected hardware profile.

```
nself ai local models recommend [flags]
```

**Example**

```bash
$ nself ai local models recommend
Detected: Apple Silicon, 32 GB unified memory, no discrete GPU

Recommended models:
  ✅ llama3.1:8b        (needs 8 GB — comfortable)
  ✅ qwen2.5-coder:7b   (needs 6 GB — comfortable)
  🟡 mistral-nemo       (needs 12 GB — marginal, may page)
  🔒 llama3.1:70b       (needs 40 GB — insufficient)
```

---

## Hardware compatibility marks

| Mark | Meaning |
|------|---------|
| ✅ | Host RAM/VRAM comfortably meets requirements |
| 🟡 | Host meets requirements marginally; model may page to swap |
| 🔒 | Insufficient RAM or VRAM to run the model |
| ☁ | Cloud-only provider; no hardware requirement |

Apple Silicon: unified memory is counted toward both RAM and VRAM with a 1.5× comfort threshold applied before emitting a ✅.

---

## Model registry

The registry (`models/registry.yml` in the AI plugin) covers:

| Provider | Models |
|----------|--------|
| Anthropic | claude-opus-4-7, claude-sonnet-4-6, claude-haiku-4-5 |
| Google | gemini-2.5-pro, gemini-2.5-flash, gemini-2.5-flash-lite |
| OpenAI | gpt-4o, gpt-4o-mini |
| Ollama | llama3.1:8b, llama3.1:70b, mistral-nemo, qwen2.5-coder:7b |

The registry ships with the plugin binary. It is never fetched at runtime, so `nself ai local models list` works offline for metadata. Pulling a model requires network access.

---

## See also

- [`nself ai local install`](cmd-ai.md#ai-local-install), Install Ollama runtime
- [`nself ai local health`](cmd-ai.md#ai-local-health), Check Ollama and plugin health
- [`nself-ai-gateway`](nself-ai-gateway.md), AI gateway plugin reference (provider keys, routing, quota)
