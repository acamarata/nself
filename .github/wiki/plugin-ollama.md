# Plugin: ollama

Local LLM provider for nSelf. Runs an [Ollama](https://ollama.com) Docker container that serves models locally and registers itself as an AI provider so all nSelf AI features can run offline, without an API key, and without sending data to any cloud service.

## How it works

This is a **config-type plugin** — it orchestrates a Docker container via `compose_fragment.yml`. There is no separate nSelf Go service binary. The nSelf CLI injects the compose fragment during `nself build`, and the Ollama container runs alongside your other services.

On first start, the plugin auto-pulls `gemma-3-4b` unless you set a different default model via `NSELF_OLLAMA_DEFAULT_MODEL`.

## Install

```bash
nself plugin install ollama
nself build
nself start
```

After `nself start`, the Ollama API is available at `http://127.0.0.1:11434`.

## Configuration

| Env var | Required | Default | Description |
|---|---|---|---|
| `NSELF_AI_PROVIDER` | No | `openai` | Set to `ollama` to route all AI calls through this plugin |
| `NSELF_OLLAMA_HOST` | No | `http://127.0.0.1:11434` | Internal URL where Ollama is reachable |
| `NSELF_OLLAMA_DEFAULT_MODEL` | No | `gemma-3-4b` | Model pulled on first start and used as default |
| `NSELF_OLLAMA_AUTO_PULL` | No | `true` | Automatically pull the default model on container start |
| `NSELF_OLLAMA_GPU` | No | `false` | Enable GPU passthrough (see GPU section below) |
| `NSELF_OLLAMA_CONTEXT_WINDOW` | No | `8192` | Context window size in tokens |
| `NSELF_OLLAMA_TIMEOUT_SECONDS` | No | `120` | Request timeout for model inference |
| `OLLAMA_ENABLED` | No | — | Set to `true` by the provider registration step |
| `PLUGIN_AI_OLLAMA_URL` | No | — | Mapped from `NSELF_OLLAMA_HOST` for plugin-ai |
| `PLUGIN_AI_OLLAMA_MODEL` | No | — | Mapped from `NSELF_OLLAMA_DEFAULT_MODEL` for plugin-ai |

## Compose mode

The plugin adds an `ollama` service to your Docker Compose stack. Key details:

- Image: `ollama/ollama:latest`
- Models stored in a named Docker volume: `{PROJECT}_ollama_models`
- Health check: `curl -sf http://localhost:11434/api/version`
- Always binds port 11434 to `127.0.0.1` — never exposed on a public interface

## Provider registration

When `NSELF_AI_PROVIDER=ollama` is set, the plugin registers itself with plugin-ai via environment variable mapping:

```
NSELF_OLLAMA_HOST      → PLUGIN_AI_OLLAMA_URL
NSELF_OLLAMA_DEFAULT_MODEL → PLUGIN_AI_OLLAMA_MODEL
OLLAMA_ENABLED=true
```

All AI features in nSelf (ɳClaw, cron AI steps, content summarization) then route through your local Ollama instance.

## GPU passthrough

GPU support is opt-in and strictly localhost-only.

**Enable GPU:**

```bash
# In your .env.local or .env.secrets:
NSELF_OLLAMA_GPU=true

nself build   # regenerates docker-compose with GPU block
nself restart ollama
```

When `NSELF_OLLAMA_GPU=true`, `nself build` injects the NVIDIA GPU passthrough block into the Ollama service definition:

```yaml
deploy:
  resources:
    reservations:
      devices:
        - driver: nvidia
          count: all
          capabilities: [gpu]
```

**Security notes:**

- GPU passthrough does NOT add any network surface. Port 11434 remains bound to `127.0.0.1`.
- NVIDIA CUDA or AMD ROCm drivers must be installed separately on the host.
- The `ollama/ollama:latest` image handles GPU detection internally.
- ROCm support: use `ollama/ollama:rocm` instead by overriding the image in your compose override file.

## Model management

Pull additional models while the container is running:

```bash
# Via Docker exec:
docker exec -it {PROJECT}-ollama ollama pull llama3.2

# Or use the Ollama API directly:
curl http://127.0.0.1:11434/api/pull -d '{"name": "llama3.2"}'

# List installed models:
curl http://127.0.0.1:11434/api/tags
```

Models are persisted in the `nself_ollama_models` Docker volume and survive container restarts.

## Database table

| Table | Purpose |
|---|---|
| `np_ollama_model_registry` | Tracks models pulled, their sizes, and last-used timestamps |

## Troubleshooting

**Container doesn't start:** check that Docker is running and you have at least 2 GB of free memory (`NSELF_OLLAMA_MIN_MEMORY_MB=2048`).

**Model pull hangs:** large models (>4 GB) can take several minutes on first pull. Check progress with `docker logs {PROJECT}-ollama`.

**AI plugin not using Ollama:** ensure `NSELF_AI_PROVIDER=ollama` is set and you've run `nself build` + `nself restart`.

## See also

- [plugin-ai](plugin-ai.md) — the AI plugin that consumes this provider
- [Ollama model library](https://ollama.com/library) — available models
