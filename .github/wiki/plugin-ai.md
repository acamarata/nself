# AI Plugin

> Multi-provider AI inference service with prompt marketplace and consensus mode. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install ai
```

## What It Does

Runs a unified AI inference service supporting 7 providers: OpenAI, Anthropic, Gemini, Groq, Mistral, Ollama, and custom endpoints. Implements a 3-layer request pipeline (pre-processing → inference → post-processing), prompt marketplace for sharing and reusing prompts, consensus mode for multi-model agreement, and shadow testing to compare providers without affecting production. Required by the `claw`, `mux`, and `voice` plugins.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `AI_PORT` | `3709` | AI service port |
| `AI_DEFAULT_PROVIDER` | `openai` | Default inference provider |
| `OPENAI_API_KEY` | — | OpenAI API key |
| `ANTHROPIC_API_KEY` | — | Anthropic API key |
| `GEMINI_API_KEY` | — | Google Gemini API key |
| `GROQ_API_KEY` | — | Groq API key |
| `MISTRAL_API_KEY` | — | Mistral API key |
| `OLLAMA_HOST` | — | Ollama server URL (for local models) |
| `AI_CONSENSUS_THRESHOLD` | `0.7` | Agreement threshold for consensus mode |

## Ports

| Port | Purpose |
|------|---------|
| 3709 | AI inference REST API |

## Database Tables

6 tables added to your Postgres database:
- `np_ai_providers` — provider configurations and status
- `np_ai_prompts` — prompt marketplace entries
- `np_ai_requests` — inference request log
- `np_ai_responses` — response cache and audit log
- `np_ai_pipeline_configs` — 3-layer pipeline configurations
- `np_ai_shadow_tests` — shadow testing results

## Nginx Routes

| Route | Target |
|-------|--------|
| `/ai/` | AI inference API |

## API

```
GET  /health                — Health check
POST /chat                  — Chat completion (OpenAI-compatible)
POST /complete              — Text completion
POST /embed                 — Embeddings
GET  /providers             — List available providers
GET  /models                — List available models
POST /prompts               — Save to prompt marketplace
GET  /prompts               — Browse marketplace
```

## Used By

- `claw` — AI agent reasoning
- `mux` — Email/message classification
- `voice` — Speech transcription and synthesis

## Gemini OAuth

The AI plugin supports a Gemini Free Pool (GFP) for routing requests across multiple OAuth-connected Google accounts. If a token expires or a scope change occurs, re-authorization is required. See [[Plugins-AI-OAuth]] for the full re-auth procedure, token rotation details, and troubleshooting.
